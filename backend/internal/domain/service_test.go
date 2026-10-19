package domain

import (
	"testing"
)

func fixture(t *testing.T) (*Service, User, User, *MemoryNotifier) {
	t.Helper()
	notifier := &MemoryNotifier{}
	service, err := NewService(SchoolSettings{ID: "school-1", Name: "Elimu High", Mode: MixedMode}, notifier)
	if err != nil {
		t.Fatal(err)
	}
	classTeacher := User{ID: "teacher-1", SchoolID: "school-1", Name: "Class Teacher", Roles: []Role{RoleClassTeacher}}
	parent := User{ID: "parent-1", SchoolID: "school-1", Name: "Parent", Phone: "+254700000001", Roles: []Role{RoleParent}}
	for _, user := range []User{classTeacher, parent} {
		if err := service.AddUser(user); err != nil {
			t.Fatal(err)
		}
	}
	if err := service.AddStudent(Student{ID: "student-1", SchoolID: "school-1", AdmissionNumber: "A001", Name: "Amina", ClassStream: "Grade 10 A", ParentIDs: []string{"parent-1"}}); err != nil {
		t.Fatal(err)
	}
	if err := service.AddStudent(Student{ID: "student-2", SchoolID: "school-1", AdmissionNumber: "A002", Name: "Brian", ClassStream: "Grade 10 A"}); err != nil {
		t.Fatal(err)
	}
	return service, classTeacher, parent, notifier
}

func TestAttendanceLifecycleEscalatesAndAlertsParent(t *testing.T) {
	service, classTeacher, _, notifier := fixture(t)
	date := "2026-10-12"
	if err := service.RecordMorningRoll(classTeacher, date, "Grade 10 A", map[string]AttendanceStatus{"student-1": Present, "student-2": Present}); err != nil {
		t.Fatal(err)
	}
	lessonTeacher := User{ID: "lesson-1", SchoolID: "school-1", Name: "Subject Teacher", Roles: []Role{RoleLessonTeacher}}
	if err := service.RecordLesson(lessonTeacher, date, "Grade 10 A", "Mathematics", map[string]AttendanceStatus{"student-1": Absent, "student-2": Present}); err != nil {
		t.Fatal(err)
	}
	reconciliation, err := service.Reconcile(classTeacher, date, "Grade 10 A", map[string]string{"student-1": "unauthorized"})
	if err != nil {
		t.Fatal(err)
	}
	if len(reconciliation.EscalatedIDs) != 1 || len(notifier.Sent) != 1 {
		t.Fatalf("expected one escalation and SMS, got %#v and %#v", reconciliation, notifier.Sent)
	}
}

func TestReconcileRequiresReason(t *testing.T) {
	service, classTeacher, _, _ := fixture(t)
	if err := service.RecordMorningRoll(classTeacher, "2026-10-12", "Grade 10 A", map[string]AttendanceStatus{"student-1": Present, "student-2": Present}); err != nil {
		t.Fatal(err)
	}
	lessonTeacher := User{ID: "lesson-1", SchoolID: "school-1", Name: "Subject Teacher", Roles: []Role{RoleLessonTeacher}}
	if err := service.RecordLesson(lessonTeacher, "2026-10-12", "Grade 10 A", "English", map[string]AttendanceStatus{"student-1": Absent}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Reconcile(classTeacher, "2026-10-12", "Grade 10 A", map[string]string{}); err == nil {
		t.Fatal("expected missing reason error")
	}
}

func TestPrincipalRoleUnionCanUpdateSettings(t *testing.T) {
	service, _, _, _ := fixture(t)
	principal := User{ID: "principal", SchoolID: "school-1", Name: "Principal", Roles: []Role{RolePrincipal, RoleClassTeacher}}
	if err := service.UpdateSettings(principal, SchoolSettings{ID: "school-1", Name: "Elimu High", Mode: Boarding}); err != nil {
		t.Fatal(err)
	}
	if service.Settings().Mode != Boarding {
		t.Fatal("expected boarding mode")
	}
}

func TestStudentVisibilityIsTenantAndParentScoped(t *testing.T) {
	service, _, parent, _ := fixture(t)
	if !service.CanAccessStudent(parent, "student-1") {
		t.Fatal("linked parent should access the student's fee and results summary")
	}
	if service.CanAccessStudent(parent, "student-2") {
		t.Fatal("parent must not access an unrelated student")
	}
	outsider := User{ID: "outsider", SchoolID: "another-school", Roles: []Role{RoleParent}}
	if service.CanAccessStudent(outsider, "student-1") {
		t.Fatal("cross-tenant access must be denied")
	}
}

func TestAuthenticateAcceptsEmailAndPhoneIdentifiers(t *testing.T) {
	service, _, _, _ := fixture(t)
	registrar := User{ID: "registrar", SchoolID: "school-1", Name: "Registrar", Email: "registrar@example.test", Phone: "+254700000099", Roles: []Role{RolePrincipal}}
	if err := service.AddUser(registrar); err != nil {
		t.Fatal(err)
	}
	if err := service.SetPassword(registrar, registrar.ID, "SecurePass1"); err != nil {
		t.Fatal(err)
	}
	for _, identifier := range []string{registrar.ID, registrar.Email, registrar.Phone} {
		user, err := service.Authenticate(identifier, "SecurePass1")
		if err != nil || user.ID != registrar.ID {
			t.Fatalf("login with %q failed: %#v %v", identifier, user, err)
		}
	}
}

func TestCBCEnrollmentAndCompetencyAssessment(t *testing.T) {
	service, _, _, _ := fixture(t)
	dos := User{ID: "dos", SchoolID: "school-1", Name: "DOS", Roles: []Role{RoleDOS}}
	if err := service.EnrollCBC(dos, CBCEnrollment{StudentID: "student-1", Grade: 10, Pathway: PathwaySTEM, Electives: []string{"Computer Science"}}); err != nil {
		t.Fatal(err)
	}
	if err := service.RecordCompetencyAssessment(dos, CompetencyAssessment{
		StudentID: "student-1", Subject: "Mathematics", TermID: "term-1", Level: MeetingExpectations,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSubmittedCBCExamSheetIsLockedAndVerificationIsAdmissionSorted(t *testing.T) {
	service, _, _, _ := fixture(t)
	teacher := User{ID: "lesson-1", SchoolID: "school-1", Name: "Teacher", Roles: []Role{RoleLessonTeacher}}
	sheet, err := service.CreateExamSheet(teacher, "Grade 10 A", "Mathematics")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetExamMark(teacher, sheet.ID, "student-1", 82); err != nil {
		t.Fatal(err)
	}
	if err := service.SetExamMark(teacher, sheet.ID, "student-2", 74); err != nil {
		t.Fatal(err)
	}
	if err := service.SubmitExamSheet(teacher, sheet.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.SetExamMark(teacher, sheet.ID, "student-1", 99); err == nil {
		t.Fatal("expected submitted sheet to reject mark changes")
	}
	examOfficer := User{ID: "exam", SchoolID: "school-1", Name: "Exam Officer", Roles: []Role{RoleExamOfficer}}
	list, err := service.VerificationList(examOfficer, sheet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].AdmissionNumber != "A001" || list[1].AdmissionNumber != "A002" {
		t.Fatalf("expected admission-sorted verification list, got %#v", list)
	}
}

func TestCorrectionRequiresExamOfficerAndAuditsChange(t *testing.T) {
	service, _, _, _ := fixture(t)
	teacher := User{ID: "lesson-1", SchoolID: "school-1", Name: "Teacher", Roles: []Role{RoleLessonTeacher}}
	sheet, err := service.CreateExamSheet(teacher, "Grade 10 A", "English")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetExamMark(teacher, sheet.ID, "student-1", 60); err != nil {
		t.Fatal(err)
	}
	if err := service.SubmitExamSheet(teacher, sheet.ID); err != nil {
		t.Fatal(err)
	}
	ticket, err := service.RequestCorrection(teacher, sheet.ID, "student-1", 68, "Verified transcription error")
	if err != nil {
		t.Fatal(err)
	}
	officer := User{ID: "exam", SchoolID: "school-1", Name: "Exam Officer", Roles: []Role{RoleExamOfficer}}
	if err := service.DecideCorrection(officer, ticket.ID, true); err != nil {
		t.Fatal(err)
	}
	log, err := service.AuditLog(User{ID: "principal", SchoolID: "school-1", Name: "Principal", Roles: []Role{RolePrincipal}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range log {
		if event.Entity == "correction_ticket" && event.EntityID == ticket.ID && event.Action == "exam.correction_decided" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected immutable correction decision audit event")
	}
}
