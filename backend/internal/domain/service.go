package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type SchoolMode string

const (
	DayOnly   SchoolMode = "day"
	Boarding  SchoolMode = "boarding"
	MixedMode SchoolMode = "mixed"
)

type Role string

const (
	RolePrincipal     Role = "principal"
	RoleClassTeacher  Role = "class_teacher"
	RoleLessonTeacher Role = "lesson_teacher"
	RoleDeputy        Role = "deputy_discipline"
	RoleParent        Role = "parent"
	RoleRegistrar     Role = "registrar"
	RolePrefect       Role = "prefect"
	RoleNurse         Role = "nurse"
	RoleHOS           Role = "head_of_subject"
	RoleExamOfficer   Role = "exam_officer"
	RoleDOS           Role = "director_of_studies"
	RoleBursar        Role = "bursar"
	RoleAccountsClerk Role = "accounts_clerk"
)

type SchoolSettings struct {
	ID                    string     `json:"id"`
	Name                  string     `json:"name"`
	Mode                  SchoolMode `json:"mode"`
	PublicSchool          bool       `json:"public_school"`
	FeeWithholdingEnabled bool       `json:"fee_withholding_enabled"`
}

type User struct {
	ID       string `json:"id"`
	SchoolID string `json:"school_id"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Roles    []Role `json:"roles"`
}

type Student struct {
	ID                 string   `json:"id"`
	SchoolID           string   `json:"school_id"`
	AdmissionNumber    string   `json:"admission_number"`
	Name               string   `json:"name"`
	NEMISUPI           string   `json:"nemis_upi"`
	BirthCertificateNo string   `json:"birth_certificate_no"`
	ClassStream        string   `json:"class_stream"`
	ParentIDs          []string `json:"parent_ids"`
	MedicalConditions  string   `json:"medical_conditions"`
}

type Staff struct {
	ID             string   `json:"id"`
	SchoolID       string   `json:"school_id"`
	Name           string   `json:"name"`
	TSCNumber      string   `json:"tsc_number"`
	EmploymentType string   `json:"employment_type"`
	Subjects       []string `json:"subjects"`
}

type AttendanceStatus string

const (
	Present AttendanceStatus = "present"
	Absent  AttendanceStatus = "absent"
	Late    AttendanceStatus = "late"
	Excused AttendanceStatus = "excused"
)

type MorningRoll struct {
	Date        string                      `json:"date"`
	ClassStream string                      `json:"class_stream"`
	RecordedBy  string                      `json:"recorded_by"`
	Students    map[string]AttendanceStatus `json:"students"`
}

type LessonAttendance struct {
	ID          string                      `json:"id"`
	Date        string                      `json:"date"`
	ClassStream string                      `json:"class_stream"`
	Subject     string                      `json:"subject"`
	TeacherID   string                      `json:"teacher_id"`
	Students    map[string]AttendanceStatus `json:"students"`
}

type Reconciliation struct {
	Date         string            `json:"date"`
	ClassStream  string            `json:"class_stream"`
	Reasons      map[string]string `json:"reasons"`
	EscalatedIDs []string          `json:"escalated_student_ids"`
}

type SMS struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

type Notifier interface {
	SendSMS(to, message string) error
}

type MemoryNotifier struct {
	mu   sync.Mutex
	Sent []SMS
}

func (n *MemoryNotifier) SendSMS(to, message string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Sent = append(n.Sent, SMS{To: to, Message: message})
	return nil
}

type Service struct {
	mu              sync.RWMutex
	settings        SchoolSettings
	users           map[string]User
	students        map[string]Student
	staff           map[string]Staff
	morning         map[string]MorningRoll
	lessons         map[string]LessonAttendance
	reconciliations map[string]Reconciliation
	notifier        Notifier
	terms           map[string]AcademicTerm
	streams         map[string]Stream
	accounts        map[string]Account
	leaves          map[string]LeaveApplication
	counts          map[string]PrefectCount
	deliveries      map[string]LessonDelivery
	audits          []AuditEvent
	cbcEnrollments  map[string]CBCEnrollment
	examSheets      map[string]ExamSheet
	corrections     map[string]CorrectionTicket
	voteHeads       map[string]VoteHead
	invoices        map[string]Invoice
	ledger          []LedgerEntry
	payments        map[string]Payment
	unmatched       map[string]Payment
	waivers         map[string]FeeWaiver
}

func NewService(settings SchoolSettings, notifier Notifier) (*Service, error) {
	if settings.ID == "" || settings.Name == "" {
		return nil, errors.New("school id and name are required")
	}
	if settings.Mode != DayOnly && settings.Mode != Boarding && settings.Mode != MixedMode {
		return nil, fmt.Errorf("invalid school mode %q", settings.Mode)
	}
	if notifier == nil {
		return nil, errors.New("notifier is required")
	}
	return &Service{
		settings: settings, users: map[string]User{}, students: map[string]Student{},
		staff: map[string]Staff{}, morning: map[string]MorningRoll{},
		lessons: map[string]LessonAttendance{}, reconciliations: map[string]Reconciliation{},
		terms: map[string]AcademicTerm{}, streams: map[string]Stream{}, accounts: map[string]Account{},
		leaves: map[string]LeaveApplication{}, counts: map[string]PrefectCount{}, deliveries: map[string]LessonDelivery{},
		cbcEnrollments: map[string]CBCEnrollment{}, examSheets: map[string]ExamSheet{}, corrections: map[string]CorrectionTicket{},
		voteHeads: map[string]VoteHead{}, invoices: map[string]Invoice{}, payments: map[string]Payment{},
		unmatched: map[string]Payment{}, waivers: map[string]FeeWaiver{},
		notifier: notifier,
	}, nil
}

func (s *Service) Settings() SchoolSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *Service) UpdateSettings(actor User, settings SchoolSettings) error {
	if !hasRole(actor, RolePrincipal) {
		return errors.New("only a principal can update school settings")
	}
	if settings.ID != s.settings.ID || settings.Name == "" {
		return errors.New("school identity cannot be changed")
	}
	if settings.Mode != DayOnly && settings.Mode != Boarding && settings.Mode != MixedMode {
		return fmt.Errorf("invalid school mode %q", settings.Mode)
	}
	s.mu.Lock()
	s.settings = settings
	s.audit(actor, "school.settings_updated", "school", settings.ID, string(settings.Mode))
	s.mu.Unlock()
	return nil
}

func (s *Service) AddUser(user User) error {
	if user.ID == "" || user.SchoolID != s.settings.ID || user.Name == "" || len(user.Roles) == 0 {
		return errors.New("user id, school, name, and at least one role are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[user.ID]; exists {
		return errors.New("user already exists")
	}
	s.users[user.ID] = user
	s.audit(user, "user.created", "user", user.ID, strings.Join(roleNames(user.Roles), ","))
	return nil
}

func (s *Service) User(id string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *Service) AddStudent(student Student) error {
	if student.ID == "" || student.SchoolID != s.settings.ID || student.Name == "" || student.AdmissionNumber == "" || student.ClassStream == "" {
		return errors.New("student id, school, name, admission number, and class stream are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.students[student.ID]; exists {
		return errors.New("student already exists")
	}
	s.students[student.ID] = student
	s.audit(User{ID: "system", SchoolID: student.SchoolID}, "student.created", "student", student.ID, student.AdmissionNumber)
	return nil
}

func (s *Service) AddStaff(staff Staff) error {
	if staff.ID == "" || staff.SchoolID != s.settings.ID || staff.Name == "" {
		return errors.New("staff id, school, and name are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.staff[staff.ID]; exists {
		return errors.New("staff already exists")
	}
	s.staff[staff.ID] = staff
	s.audit(User{ID: "system", SchoolID: staff.SchoolID}, "staff.created", "staff", staff.ID, staff.TSCNumber)
	return nil
}

func (s *Service) RecordMorningRoll(actor User, date, classStream string, statuses map[string]AttendanceStatus) error {
	if !hasRole(actor, RoleClassTeacher) {
		return errors.New("only a class teacher can record the morning roll")
	}
	if date == "" || classStream == "" || len(statuses) == 0 {
		return errors.New("date, class stream, and students are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, status := range statuses {
		student, ok := s.students[id]
		if !ok || student.ClassStream != classStream || (status != Present && status != Absent && status != Late && status != Excused) {
			return fmt.Errorf("invalid morning roll entry for student %q", id)
		}
	}
	key := date + "|" + classStream
	s.morning[key] = MorningRoll{Date: date, ClassStream: classStream, RecordedBy: actor.ID, Students: cloneStatuses(statuses)}
	s.audit(actor, "attendance.morning_roll", "roll", key, fmt.Sprintf("students=%d", len(statuses)))
	return nil
}

func (s *Service) RecordLesson(actor User, date, classStream, subject string, statuses map[string]AttendanceStatus) error {
	if !hasRole(actor, RoleLessonTeacher) {
		return errors.New("only a lesson teacher can record lesson attendance")
	}
	key := date + "|" + classStream
	s.mu.Lock()
	defer s.mu.Unlock()
	morning, ok := s.morning[key]
	if !ok {
		return errors.New("morning roll must be recorded before lesson attendance")
	}
	for id, status := range statuses {
		if _, inMorning := morning.Students[id]; !inMorning || (status != Present && status != Absent && status != Late && status != Excused) {
			return fmt.Errorf("student %q is not in the morning roll", id)
		}
	}
	id := randomID()
	s.lessons[id] = LessonAttendance{ID: id, Date: date, ClassStream: classStream, Subject: subject, TeacherID: actor.ID, Students: cloneStatuses(statuses)}
	s.audit(actor, "attendance.lesson_recorded", "lesson", id, subject)
	return nil
}

func (s *Service) Reconcile(actor User, date, classStream string, reasons map[string]string) (Reconciliation, error) {
	if !hasRole(actor, RoleClassTeacher) {
		return Reconciliation{}, errors.New("only a class teacher can reconcile attendance")
	}
	key := date + "|" + classStream
	s.mu.Lock()
	defer s.mu.Unlock()
	morning, ok := s.morning[key]
	if !ok {
		return Reconciliation{}, errors.New("morning roll not found")
	}
	missed := map[string]bool{}
	for _, lesson := range s.lessons {
		if lesson.Date != date || lesson.ClassStream != classStream {
			continue
		}
		for id, status := range lesson.Students {
			if status == Absent && morning.Students[id] == Present {
				missed[id] = true
			}
		}
	}
	for id := range missed {
		if strings.TrimSpace(reasons[id]) == "" {
			return Reconciliation{}, fmt.Errorf("absence reason is required for student %q", id)
		}
	}
	result := Reconciliation{Date: date, ClassStream: classStream, Reasons: map[string]string{}, EscalatedIDs: []string{}}
	for id := range missed {
		result.Reasons[id] = reasons[id]
		if reasons[id] == "unauthorized" {
			result.EscalatedIDs = append(result.EscalatedIDs, id)
			student := s.students[id]
			for _, parentID := range student.ParentIDs {
				if parent, exists := s.users[parentID]; exists && parent.Phone != "" {
					if err := s.notifier.SendSMS(parent.Phone, fmt.Sprintf("%s was absent from a lesson without approved leave.", student.Name)); err != nil {
						return Reconciliation{}, fmt.Errorf("send absence alert for %s: %w", id, err)
					}
				}
			}
		}
	}
	sort.Strings(result.EscalatedIDs)
	s.reconciliations[key] = result
	s.audit(actor, "attendance.reconciled", "reconciliation", key, fmt.Sprintf("escalated=%d", len(result.EscalatedIDs)))
	return result, nil
}

func hasRole(user User, wanted Role) bool {
	for _, role := range user.Roles {
		if role == wanted || role == RolePrincipal {
			return true
		}
	}
	return false
}

func cloneStatuses(input map[string]AttendanceStatus) map[string]AttendanceStatus {
	output := make(map[string]AttendanceStatus, len(input))
	for id, status := range input {
		output[id] = status
	}
	return output
}

func roleNames(roles []Role) []string {
	names := make([]string, len(roles))
	for i, role := range roles {
		names[i] = string(role)
	}
	return names
}

func randomID() string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", bytes)
}
