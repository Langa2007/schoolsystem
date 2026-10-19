package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// These workflow records form the Phase One domain model. The current service
// stores them in memory; backend/migrations defines their durable SQL schema.
type AcademicTerm struct {
	ID, SchoolID, Name string
	Year               int
	Opens, Closes      time.Time
	Holidays           []DateRange
	Active             bool
}
type DateRange struct{ Start, End time.Time }
type Stream struct {
	ID, SchoolID, Name, Grade, ClassTeacherID, PrefectID string
	Capacity                                             int
}
type Account struct {
	UserID, Salt, PasswordHash string
	MustChangePassword, Locked bool
	FailedAttempts             int
}
type LeaveStatus string

const (
	LeavePending  LeaveStatus = "pending"
	LeaveApproved LeaveStatus = "approved"
	LeaveRejected LeaveStatus = "rejected"
)

type LeaveApplication struct {
	ID, SchoolID, StudentID, SubmittedBy, Reason string
	Start, End                                   time.Time
	Status                                       LeaveStatus
	ApprovedBy                                   string
	CreatedAt, DecidedAt                         time.Time
}
type PrefectCount struct {
	Date, ClassStream, PrefectID string
	Present, Absent              int
	SubmittedAt                  time.Time
}
type LessonDelivery struct {
	ID, Date, ClassStream, Subject, TeacherID, Topic, Status, Reason string
	ReviewStatus, ReviewedBy                                         string
}
type AuditEvent struct {
	ID, SchoolID, ActorID, Action, Entity, EntityID string
	At                                              time.Time
	Details                                         string
}

func (s *Service) audit(actor User, action, entity, entityID, details string) {
	s.audits = append(s.audits, AuditEvent{ID: randomID(), SchoolID: s.settings.ID, ActorID: actor.ID, Action: action, Entity: entity, EntityID: entityID, At: time.Now().UTC(), Details: details})
}

func (s *Service) AuditLog(actor User) ([]AuditEvent, error) {
	if !hasRole(actor, RolePrincipal) {
		return nil, errors.New("only a principal can read the audit log")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]AuditEvent(nil), s.audits...), nil
}

func (s *Service) CreateTerm(actor User, term AcademicTerm) error {
	if !hasRole(actor, RolePrincipal) && !hasRole(actor, RoleRegistrar) {
		return errors.New("registrar or principal required")
	}
	if term.ID == "" || term.Name == "" || term.Year < 2000 || !term.Opens.Before(term.Closes) {
		return errors.New("term id, name, year and valid date range are required")
	}
	for _, h := range term.Holidays {
		if h.Start.After(h.End) || h.Start.Before(term.Opens) || h.End.After(term.Closes) {
			return errors.New("holiday must be within the term")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if term.SchoolID != "" && term.SchoolID != s.settings.ID {
		return errors.New("school mismatch")
	}
	term.SchoolID = s.settings.ID
	if term.Active {
		for id, existing := range s.terms {
			if existing.Year == term.Year && existing.Active {
				existing.Active = false
				s.terms[id] = existing
			}
		}
	}
	if _, ok := s.terms[term.ID]; ok {
		return errors.New("term already exists")
	}
	s.terms[term.ID] = term
	s.audit(actor, "term.created", "term", term.ID, term.Name)
	return nil
}

func (s *Service) CreateStream(actor User, stream Stream) error {
	if !hasRole(actor, RolePrincipal) && !hasRole(actor, RoleRegistrar) {
		return errors.New("registrar or principal required")
	}
	if stream.ID == "" || stream.Name == "" || stream.Grade == "" || stream.Capacity < 1 {
		return errors.New("stream id, grade, name and positive capacity are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if stream.ClassTeacherID != "" {
		u, ok := s.users[stream.ClassTeacherID]
		if !ok || (!hasRole(u, RoleClassTeacher) && !hasRole(u, RolePrincipal)) {
			return errors.New("class teacher account not found or role missing")
		}
	}
	if stream.PrefectID != "" {
		u, ok := s.users[stream.PrefectID]
		if !ok || !hasRole(u, RolePrefect) {
			return errors.New("prefect account not found or role missing")
		}
	}
	if _, ok := s.streams[stream.ID]; ok {
		return errors.New("stream already exists")
	}
	stream.SchoolID = s.settings.ID
	s.streams[stream.ID] = stream
	s.audit(actor, "stream.created", "stream", stream.ID, stream.Name)
	return nil
}

func (s *Service) SubmitPrefectCount(actor User, date, streamID string, present, absent int) error {
	if !hasRole(actor, RolePrefect) {
		return errors.New("only an assigned prefect can submit a headcount")
	}
	if date == "" || present < 0 || absent < 0 || present+absent == 0 {
		return errors.New("valid date and non-empty non-negative headcount required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stream, ok := s.streams[streamID]
	if !ok || stream.PrefectID != actor.ID {
		return errors.New("prefect is not assigned to this stream")
	}
	key := date + "|" + stream.Name
	s.counts[key] = PrefectCount{Date: date, ClassStream: stream.Name, PrefectID: actor.ID, Present: present, Absent: absent, SubmittedAt: time.Now().UTC()}
	s.audit(actor, "attendance.prefect_count", "count", key, fmt.Sprintf("present=%d absent=%d", present, absent))
	return nil
}

func (s *Service) PostMorningRoll(actor User, date, streamID string, statuses map[string]AttendanceStatus) error {
	if !hasRole(actor, RoleClassTeacher) {
		return errors.New("only a class teacher can post the official roll")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stream, ok := s.streams[streamID]
	if !ok || stream.ClassTeacherID != actor.ID {
		return errors.New("class teacher is not assigned to this stream")
	}
	key := date + "|" + stream.Name
	if _, ok := s.counts[key]; !ok {
		return errors.New("prefect headcount must be submitted first")
	}
	if len(statuses) == 0 {
		return errors.New("student statuses are required")
	}
	for id, status := range statuses {
		st, found := s.students[id]
		if !found || st.ClassStream != stream.Name {
			return fmt.Errorf("student %q does not belong to stream", id)
		}
		if status != Present && status != Absent {
			return fmt.Errorf("invalid status for student %q", id)
		}
	}
	s.morning[key] = MorningRoll{Date: date, ClassStream: stream.Name, RecordedBy: actor.ID, Students: cloneStatuses(statuses)}
	for id, status := range statuses {
		if status != Absent {
			continue
		}
		st := s.students[id]
		if s.hasApprovedLeave(id, date) {
			continue
		}
		for _, parentID := range st.ParentIDs {
			if parent, found := s.users[parentID]; found && parent.Phone != "" {
				if err := s.notifier.SendSMS(parent.Phone, fmt.Sprintf("%s was marked absent today.", st.Name)); err != nil {
					return fmt.Errorf("send absence alert: %w", err)
				}
			}
		}
	}
	s.audit(actor, "attendance.morning_roll", "roll", key, fmt.Sprintf("students=%d", len(statuses)))
	return nil
}

func (s *Service) hasApprovedLeave(studentID, date string) bool {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	for _, l := range s.leaves {
		if l.StudentID == studentID && l.Status == LeaveApproved && !d.Before(l.Start) && !d.After(l.End) {
			return true
		}
	}
	return false
}

func (s *Service) RequestLeave(actor User, studentID, reason string, start, end time.Time) (LeaveApplication, error) {
	if !hasRole(actor, RoleParent) && !hasRole(actor, RoleClassTeacher) && !hasRole(actor, RoleNurse) {
		return LeaveApplication{}, errors.New("parent, class teacher or nurse required")
	}
	if strings.TrimSpace(reason) == "" || start.IsZero() || start.After(end) {
		return LeaveApplication{}, errors.New("reason and valid leave dates are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.students[studentID]
	if !ok {
		return LeaveApplication{}, errors.New("student not found")
	}
	if hasRole(actor, RoleParent) {
		linked := false
		for _, p := range st.ParentIDs {
			if p == actor.ID {
				linked = true
			}
		}
		if !linked {
			return LeaveApplication{}, errors.New("parent is not linked to this student")
		}
	}
	l := LeaveApplication{ID: randomID(), SchoolID: s.settings.ID, StudentID: studentID, SubmittedBy: actor.ID, Reason: reason, Start: start, End: end, Status: LeavePending, CreatedAt: time.Now().UTC()}
	s.leaves[l.ID] = l
	s.audit(actor, "leave.requested", "leave", l.ID, reason)
	return l, nil
}

func (s *Service) DecideLeave(actor User, id string, approve bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.leaves[id]
	if !ok {
		return errors.New("leave application not found")
	}
	if l.Status != LeavePending {
		return errors.New("leave application already decided")
	}
	days := int(l.End.Sub(l.Start).Hours()/24) + 1
	allowed := hasRole(actor, RolePrincipal) || (days <= 1 && hasRole(actor, RoleClassTeacher)) || (days >= 2 && days <= 5 && hasRole(actor, RoleDeputy))
	if !allowed {
		return errors.New("actor is not authorized for this leave duration")
	}
	if days > 5 && !hasRole(actor, RolePrincipal) {
		return errors.New("leave longer than five days requires principal approval")
	}
	if approve {
		l.Status = LeaveApproved
	} else {
		l.Status = LeaveRejected
	}
	l.ApprovedBy = actor.ID
	l.DecidedAt = time.Now().UTC()
	s.leaves[id] = l
	s.audit(actor, "leave.decided", "leave", id, string(l.Status))
	return nil
}

func (s *Service) RecordLessonDelivery(actor User, d, stream, subject, status, topic, reason string) (LessonDelivery, error) {
	if !hasRole(actor, RoleLessonTeacher) {
		return LessonDelivery{}, errors.New("lesson teacher required")
	}
	if status != "done" && status != "missed" {
		return LessonDelivery{}, errors.New("status must be done or missed")
	}
	if status == "done" && strings.TrimSpace(topic) == "" {
		return LessonDelivery{}, errors.New("topic required for completed lesson")
	}
	if status == "missed" && strings.TrimSpace(reason) == "" {
		return LessonDelivery{}, errors.New("reason required for missed lesson")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	l := LessonDelivery{ID: randomID(), Date: d, ClassStream: stream, Subject: subject, TeacherID: actor.ID, Status: status, Topic: topic, Reason: reason, ReviewStatus: "pending"}
	s.deliveries[l.ID] = l
	s.audit(actor, "lesson.delivery_recorded", "lesson", l.ID, status)
	return l, nil
}

func (s *Service) ReviewLesson(actor User, id, decision string) error {
	if !hasRole(actor, RoleHOS) {
		return errors.New("head of subject role required")
	}
	if decision != "confirmed" && decision != "queried" && decision != "rejected" {
		return errors.New("invalid review decision")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.deliveries[id]
	if !ok {
		return errors.New("lesson not found")
	}
	if l.ReviewStatus != "pending" {
		return errors.New("lesson already reviewed")
	}
	l.ReviewStatus = decision
	l.ReviewedBy = actor.ID
	s.deliveries[id] = l
	s.audit(actor, "lesson.reviewed", "lesson", id, decision)
	return nil
}

func (s *Service) EnforceReconciliationDeadline(now time.Time) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 16, 30, 0, 0, now.Location())
	if now.Before(cutoff) {
		return nil
	}
	var missing []string
	date := now.Format("2006-01-02")
	for _, stream := range s.streams {
		key := date + "|" + stream.Name
		if _, ok := s.morning[key]; ok {
			if _, done := s.reconciliations[key]; done {
				continue
			}
			if stream.ClassTeacherID != "" {
				teacher := s.users[stream.ClassTeacherID]
				for _, r := range teacher.Roles {
					if r == RoleClassTeacher {
						missing = append(missing, teacher.ID)
						break
					}
				}
				for _, u := range s.users {
					if hasRole(u, RoleDeputy) && u.Phone != "" {
						_ = s.notifier.SendSMS(u.Phone, "Attendance reconciliation overdue for "+stream.Name+".")
					}
				}
			}
		}
	}
	sort.Strings(missing)
	return missing
}

func (s *Service) SetPassword(actor User, userID, password string) error {
	if !hasRole(actor, RolePrincipal) && !hasRole(actor, RoleRegistrar) {
		return errors.New("registrar or principal required")
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	salt := randomID()
	hash := passwordDigest(salt, password)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return errors.New("user not found")
	}
	s.accounts[userID] = Account{UserID: userID, Salt: salt, PasswordHash: hash, MustChangePassword: true}
	s.audit(actor, "account.provisioned", "account", userID, "password set; change required")
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	letter, digit := false, false
	for _, r := range password {
		if r >= '0' && r <= '9' {
			digit = true
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			letter = true
		}
	}
	if !letter || !digit {
		return errors.New("password must contain a letter and a number")
	}
	return nil
}

func (s *Service) ChangePassword(actor User, currentPassword, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[actor.ID]; !ok || actor.SchoolID != s.settings.ID {
		return errors.New("account not found")
	}
	account, ok := s.accounts[actor.ID]
	if !ok || account.Locked || passwordDigest(account.Salt, currentPassword) != account.PasswordHash {
		return errors.New("current password is incorrect")
	}
	if currentPassword == newPassword {
		return errors.New("new password must differ from current password")
	}
	account.Salt = randomID()
	account.PasswordHash = passwordDigest(account.Salt, newPassword)
	account.MustChangePassword = false
	account.FailedAttempts = 0
	s.accounts[actor.ID] = account
	s.audit(actor, "account.password_changed", "account", actor.ID, "password changed by user")
	return nil
}

func (s *Service) PasswordChangeRequired(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accounts[userID].MustChangePassword
}

func (s *Service) Authenticate(identifier, password string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	identifier = strings.TrimSpace(identifier)
	var u User
	ok := false
	for _, candidate := range s.users {
		if candidate.ID == identifier || candidate.Phone == identifier || (candidate.Email != "" && strings.EqualFold(candidate.Email, identifier)) {
			if ok {
				return User{}, errors.New("ambiguous login identifier")
			}
			u, ok = candidate, true
		}
	}
	if !ok {
		return User{}, errors.New("invalid credentials")
	}
	a, ok := s.accounts[u.ID]
	if !ok || a.Locked {
		return User{}, errors.New("account unavailable")
	}
	if passwordDigest(a.Salt, password) != a.PasswordHash {
		a.FailedAttempts++
		if a.FailedAttempts >= 5 {
			a.Locked = true
		}
		s.accounts[u.ID] = a
		return User{}, errors.New("invalid credentials")
	}
	a.FailedAttempts = 0
	s.accounts[u.ID] = a
	return u, nil
}
func passwordDigest(salt, password string) string {
	sum := sha256.Sum256([]byte(salt + password))
	for i := 0; i < 100000; i++ {
		sum = sha256.Sum256(sum[:])
	}
	return hex.EncodeToString(sum[:])
}

func (s *Service) AttendanceSummary(actor User, date, stream string) (map[string]int, error) {
	if !hasRole(actor, RoleClassTeacher) && !hasRole(actor, RolePrincipal) && !hasRole(actor, RoleDeputy) {
		return nil, errors.New("attendance access denied")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	roll, ok := s.morning[date+"|"+stream]
	if !ok {
		return nil, errors.New("morning roll not found")
	}
	summary := map[string]int{"present": 0, "absent": 0, "excused": 0}
	for id, status := range roll.Students {
		if status == Absent && s.hasApprovedLeave(id, date) {
			summary["excused"]++
		} else if status == Present {
			summary["present"]++
		} else {
			summary["absent"]++
		}
	}
	return summary, nil
}
