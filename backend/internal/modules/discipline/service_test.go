package discipline

import "testing"

func TestCounselingNotesAreCounselorOnly(t *testing.T) {
	service := NewService()
	if err := service.RegisterCounselor("counselor-1"); err != nil {
		t.Fatal(err)
	}
	if err := service.AddCounselingNote(CounselingNote{ID: "note-1", SchoolID: "school-1", StudentID: "student-1", CounselorID: "counselor-1", Note: "follow-up"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CounselingNotes("teacher-1", "school-1", "student-1"); err == nil {
		t.Fatal("expected non-counselor access to be rejected")
	}
}
