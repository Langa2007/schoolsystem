package timetable

import "time"

type LessonRequest struct {
	ID, StreamID, SubjectID, TeacherID, RoomID string
	PeriodsPerWeek, Duration                   int
	RequiresLab                                bool
}

type Availability struct {
	PersonID, Day string
	Period        int
}

type Slot struct {
	Day    string
	Period int
}

type LessonPlacement struct {
	RequestID string
	Slot
	Duration int
}

type TimetableDraft struct {
	ID, SchoolID string
	Placements   []LessonPlacement
	CreatedAt    time.Time
}
