package timetable

import (
	"context"
	"testing"
)

type fakeSolver struct{ draft TimetableDraft }

func (f fakeSolver) SolveLessons(context.Context, SolverRequest) (TimetableDraft, error) {
	return f.draft, nil
}

func TestValidatorRejectsTeacherClash(t *testing.T) {
	requests := []LessonRequest{
		{ID: "math-a", StreamID: "a", TeacherID: "teacher-1", PeriodsPerWeek: 1, Duration: 1},
		{ID: "math-b", StreamID: "b", TeacherID: "teacher-1", PeriodsPerWeek: 1, Duration: 1},
	}
	service := Service{Validator: Validator{WorkingDays: []string{"mon"}, PeriodsPerDay: 8}}
	err := service.ValidateManualEdit(requests, TimetableDraft{SchoolID: "school-1", Placements: []LessonPlacement{
		{RequestID: "math-a", Slot: Slot{Day: "mon", Period: 1}, Duration: 1},
		{RequestID: "math-b", Slot: Slot{Day: "mon", Period: 1}, Duration: 1},
	}})
	if err == nil {
		t.Fatal("expected teacher clash to be rejected")
	}
}

func TestServiceRevalidatesSolverOutput(t *testing.T) {
	request := LessonRequest{ID: "math-a", StreamID: "a", TeacherID: "teacher-1", PeriodsPerWeek: 1, Duration: 1}
	service := Service{
		Solver:    fakeSolver{draft: TimetableDraft{SchoolID: "school-1", Placements: []LessonPlacement{{RequestID: "math-a", Slot: Slot{Day: "mon", Period: 1}, Duration: 1}}}},
		Validator: Validator{WorkingDays: []string{"mon"}, PeriodsPerDay: 8},
	}
	if _, err := service.Generate(context.Background(), "school-1", []LessonRequest{request}); err != nil {
		t.Fatal(err)
	}
}

func TestValidatorRejectsWrongDurationAndDuplicateRequests(t *testing.T) {
	request := LessonRequest{ID: "lab", StreamID: "a", TeacherID: "teacher-1", RoomID: "lab-1", PeriodsPerWeek: 2, Duration: 2}
	v := Validator{WorkingDays: []string{"mon"}, PeriodsPerDay: 8}
	if err := v.Validate([]LessonRequest{request, request}, TimetableDraft{Placements: []LessonPlacement{{RequestID: "lab", Slot: Slot{Day: "mon", Period: 1}, Duration: 2}}}); err == nil {
		t.Fatal("expected duplicate request to be rejected")
	}
	if err := v.Validate([]LessonRequest{request}, TimetableDraft{Placements: []LessonPlacement{{RequestID: "lab", Slot: Slot{Day: "mon", Period: 1}, Duration: 1}}}); err == nil {
		t.Fatal("expected a placement with the wrong duration to be rejected")
	}
}

func TestValidatorRejectsRoomClashAndUnavailableTeacher(t *testing.T) {
	requests := []LessonRequest{
		{ID: "a", StreamID: "a", TeacherID: "teacher-a", RoomID: "lab", PeriodsPerWeek: 1, Duration: 1},
		{ID: "b", StreamID: "b", TeacherID: "teacher-b", RoomID: "lab", PeriodsPerWeek: 1, Duration: 1},
	}
	v := Validator{WorkingDays: []string{"mon"}, PeriodsPerDay: 8}
	draft := TimetableDraft{Placements: []LessonPlacement{
		{RequestID: "a", Slot: Slot{Day: "mon", Period: 1}, Duration: 1},
		{RequestID: "b", Slot: Slot{Day: "mon", Period: 1}, Duration: 1},
	}}
	if err := v.Validate(requests, draft); err == nil {
		t.Fatal("expected shared room clash to be rejected")
	}
	v.Unavailable = map[string]map[string]map[int]bool{"teacher:teacher-a": {"mon": {1: true}}}
	if err := v.Validate(requests[:1], TimetableDraft{Placements: draft.Placements[:1]}); err == nil {
		t.Fatal("expected teacher unavailability to be rejected")
	}
}
