package boarding

import "time"

type Dormitory struct {
	ID, SchoolID, Name, Gender string
	Capacity                   int
}

type BedAllocation struct {
	ID, DormitoryID, StudentID string
	AllocatedAt, ReleasedAt    *time.Time
}

type ExeatPass struct {
	ID, StudentID, ApprovedBy, Destination, Reason string
	Departure, ExpectedReturn                      time.Time
	ReturnedAt                                     *time.Time
}

type PrepAttendance struct {
	Date            string
	DormitoryID     string
	Present, Absent []string
}
