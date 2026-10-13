package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type CBCPathway string

const (
	PathwaySTEM          CBCPathway = "stem"
	PathwaySocialScience CBCPathway = "social_sciences"
	PathwayArtsSports    CBCPathway = "arts_and_sports_science"
)

type CompetencyLevel string

const (
	ExceedingExpectations   CompetencyLevel = "EE"
	MeetingExpectations     CompetencyLevel = "ME"
	ApproachingExpectations CompetencyLevel = "AE"
	BelowExpectations       CompetencyLevel = "BE"
)

type CBCEnrollment struct {
	StudentID string     `json:"student_id"`
	Grade     int        `json:"grade"`
	Pathway   CBCPathway `json:"pathway"`
	Electives []string   `json:"electives"`
}

type CompetencyAssessment struct {
	ID          string          `json:"id"`
	StudentID   string          `json:"student_id"`
	Subject     string          `json:"subject"`
	TermID      string          `json:"term_id"`
	Level       CompetencyLevel `json:"level"`
	Portfolio   string          `json:"portfolio"`
	ProjectNote string          `json:"project_note"`
	RecordedBy  string          `json:"recorded_by"`
}

type ExamMark struct {
	StudentID string `json:"student_id"`
	Subject   string `json:"subject"`
	Score     int    `json:"score"`
}

type ExamSheetStatus string

const (
	ExamDraft     ExamSheetStatus = "draft"
	ExamSubmitted ExamSheetStatus = "submitted"
	ExamVerified  ExamSheetStatus = "verified"
	ExamPublished ExamSheetStatus = "published"
)

type ExamSheet struct {
	ID                string              `json:"id"`
	ClassStream       string              `json:"class_stream"`
	Subject           string              `json:"subject"`
	TeacherID         string              `json:"teacher_id"`
	Marks             map[string]ExamMark `json:"marks"`
	Status            ExamSheetStatus     `json:"status"`
	DOSApproved       bool                `json:"dos_approved"`
	PrincipalApproved bool                `json:"principal_approved"`
}

type VerificationMark struct {
	AdmissionNumber string `json:"admission_number"`
	StudentName     string `json:"student_name"`
	Score           int    `json:"score"`
}

type CorrectionTicket struct {
	ID            string    `json:"id"`
	ExamSheetID   string    `json:"exam_sheet_id"`
	StudentID     string    `json:"student_id"`
	Previous      int       `json:"previous_score"`
	Requested     int       `json:"requested_score"`
	Justification string    `json:"justification"`
	Status        string    `json:"status"`
	OfficerID     string    `json:"officer_id"`
	CreatedAt     time.Time `json:"created_at"`
	DecidedAt     time.Time `json:"decided_at"`
}

func (s *Service) EnrollCBC(actor User, enrollment CBCEnrollment) error {
	if !hasRole(actor, RolePrincipal) && !hasRole(actor, RoleRegistrar) && !hasRole(actor, RoleDOS) {
		return errors.New("principal, registrar, or director of studies required")
	}
	if enrollment.StudentID == "" || enrollment.Grade < 10 || enrollment.Grade > 12 {
		return errors.New("CBC enrollment requires a Grade 10, 11, or 12 student")
	}
	if enrollment.Pathway != PathwaySTEM && enrollment.Pathway != PathwaySocialScience && enrollment.Pathway != PathwayArtsSports {
		return fmt.Errorf("invalid CBC pathway %q", enrollment.Pathway)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	student, ok := s.students[enrollment.StudentID]
	if !ok {
		return errors.New("student not found")
	}
	if _, exists := s.cbcEnrollments[enrollment.StudentID]; exists {
		return errors.New("student already has a CBC enrollment")
	}
	if !strings.HasPrefix(strings.ToLower(student.ClassStream), fmt.Sprintf("grade %d", enrollment.Grade)) {
		return errors.New("student stream does not match CBC grade")
	}
	s.cbcEnrollments[enrollment.StudentID] = enrollment
	s.audit(actor, "cbc.enrolled", "cbc_enrollment", enrollment.StudentID, string(enrollment.Pathway))
	return nil
}

func (s *Service) RecordCompetencyAssessment(actor User, assessment CompetencyAssessment) error {
	if !hasRole(actor, RoleLessonTeacher) && !hasRole(actor, RoleHOS) && !hasRole(actor, RoleDOS) {
		return errors.New("lesson teacher, head of subject, or director of studies required")
	}
	if assessment.StudentID == "" || assessment.Subject == "" || assessment.TermID == "" {
		return errors.New("student, subject, and term are required")
	}
	switch assessment.Level {
	case ExceedingExpectations, MeetingExpectations, ApproachingExpectations, BelowExpectations:
	default:
		return fmt.Errorf("invalid competency level %q", assessment.Level)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cbcEnrollments[assessment.StudentID]; !ok {
		return errors.New("student is not enrolled in CBC")
	}
	assessment.ID = randomID()
	assessment.RecordedBy = actor.ID
	s.audit(actor, "cbc.assessment_recorded", "competency_assessment", assessment.ID, string(assessment.Level))
	return nil
}

func (s *Service) CreateExamSheet(actor User, classStream, subject string) (ExamSheet, error) {
	if !hasRole(actor, RoleLessonTeacher) && !hasRole(actor, RoleExamOfficer) {
		return ExamSheet{}, errors.New("lesson teacher or exam officer required")
	}
	if classStream == "" || subject == "" {
		return ExamSheet{}, errors.New("class stream and subject are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sheet := ExamSheet{ID: randomID(), ClassStream: classStream, Subject: subject, TeacherID: actor.ID, Marks: map[string]ExamMark{}, Status: ExamDraft}
	s.examSheets[sheet.ID] = sheet
	return sheet, nil
}

func (s *Service) SetExamMark(actor User, sheetID, studentID string, score int) error {
	if !hasRole(actor, RoleLessonTeacher) && !hasRole(actor, RoleExamOfficer) {
		return errors.New("lesson teacher or exam officer required")
	}
	if score < 0 || score > 100 {
		return errors.New("score must be between 0 and 100")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sheet, ok := s.examSheets[sheetID]
	if !ok {
		return errors.New("exam sheet not found")
	}
	if sheet.Status != ExamDraft {
		return errors.New("submitted exam sheets are locked")
	}
	student, ok := s.students[studentID]
	if !ok || student.ClassStream != sheet.ClassStream {
		return errors.New("student does not belong to the exam stream")
	}
	sheet.Marks[studentID] = ExamMark{StudentID: studentID, Subject: sheet.Subject, Score: score}
	s.examSheets[sheetID] = sheet
	return nil
}

func (s *Service) SubmitExamSheet(actor User, sheetID string) error {
	if !hasRole(actor, RoleLessonTeacher) && !hasRole(actor, RoleExamOfficer) {
		return errors.New("lesson teacher or exam officer required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sheet, ok := s.examSheets[sheetID]
	if !ok {
		return errors.New("exam sheet not found")
	}
	if sheet.TeacherID != actor.ID && !hasRole(actor, RoleExamOfficer) {
		return errors.New("only the submitting teacher or exam officer can submit")
	}
	if len(sheet.Marks) == 0 {
		return errors.New("cannot submit an empty exam sheet")
	}
	if sheet.Status != ExamDraft {
		return errors.New("exam sheet is already submitted")
	}
	sheet.Status = ExamSubmitted
	s.examSheets[sheetID] = sheet
	s.audit(actor, "exam.sheet_submitted", "exam_sheet", sheetID, "marks locked")
	return nil
}

func (s *Service) VerificationList(actor User, sheetID string) ([]VerificationMark, error) {
	if !hasRole(actor, RoleExamOfficer) && !hasRole(actor, RoleDOS) && !hasRole(actor, RolePrincipal) {
		return nil, errors.New("exam verification access denied")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	sheet, ok := s.examSheets[sheetID]
	if !ok || sheet.Status == ExamDraft {
		return nil, errors.New("submitted exam sheet not found")
	}
	result := make([]VerificationMark, 0, len(sheet.Marks))
	for id, mark := range sheet.Marks {
		student := s.students[id]
		result = append(result, VerificationMark{AdmissionNumber: student.AdmissionNumber, StudentName: student.Name, Score: mark.Score})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].AdmissionNumber < result[j].AdmissionNumber })
	return result, nil
}

func (s *Service) ApproveExam(actor User, sheetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sheet, ok := s.examSheets[sheetID]
	if !ok || sheet.Status == ExamDraft {
		return errors.New("submitted exam sheet not found")
	}
	switch {
	case hasRole(actor, RoleDOS) && !sheet.DOSApproved:
		sheet.DOSApproved = true
		sheet.Status = ExamVerified
	case hasRole(actor, RolePrincipal) && sheet.DOSApproved:
		sheet.PrincipalApproved = true
		sheet.Status = ExamPublished
	default:
		return errors.New("approval gate requires DOS then principal")
	}
	s.examSheets[sheetID] = sheet
	s.audit(actor, "exam.sheet_approved", "exam_sheet", sheetID, string(sheet.Status))
	return nil
}

func (s *Service) RequestCorrection(actor User, sheetID, studentID string, requested int, justification string) (CorrectionTicket, error) {
	if strings.TrimSpace(justification) == "" || requested < 0 || requested > 100 {
		return CorrectionTicket{}, errors.New("valid requested score and justification are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sheet, ok := s.examSheets[sheetID]
	if !ok || sheet.Status == ExamDraft {
		return CorrectionTicket{}, errors.New("submitted exam sheet not found")
	}
	mark, ok := sheet.Marks[studentID]
	if !ok {
		return CorrectionTicket{}, errors.New("student mark not found")
	}
	ticket := CorrectionTicket{ID: randomID(), ExamSheetID: sheetID, StudentID: studentID, Previous: mark.Score, Requested: requested, Justification: justification, Status: "pending", CreatedAt: time.Now().UTC()}
	s.corrections[ticket.ID] = ticket
	s.audit(actor, "exam.correction_requested", "correction_ticket", ticket.ID, fmt.Sprintf("previous=%d requested=%d", mark.Score, requested))
	return ticket, nil
}

func (s *Service) DecideCorrection(actor User, ticketID string, approve bool) error {
	if !hasRole(actor, RoleExamOfficer) {
		return errors.New("exam officer required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ticket, ok := s.corrections[ticketID]
	if !ok || ticket.Status != "pending" {
		return errors.New("pending correction ticket not found")
	}
	ticket.OfficerID, ticket.DecidedAt = actor.ID, time.Now().UTC()
	if approve {
		sheet := s.examSheets[ticket.ExamSheetID]
		mark := sheet.Marks[ticket.StudentID]
		if mark.Score != ticket.Previous {
			return errors.New("mark changed since correction was requested")
		}
		mark.Score = ticket.Requested
		sheet.Marks[ticket.StudentID], ticket.Status = mark, "approved"
		s.examSheets[ticket.ExamSheetID] = sheet
	} else {
		ticket.Status = "rejected"
	}
	s.corrections[ticketID] = ticket
	s.audit(actor, "exam.correction_decided", "correction_ticket", ticketID, fmt.Sprintf("status=%s previous=%d requested=%d justification=%s", ticket.Status, ticket.Previous, ticket.Requested, ticket.Justification))
	return nil
}
