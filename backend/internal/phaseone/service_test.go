package phaseone

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
