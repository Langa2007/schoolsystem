package discipline

import "time"

type Severity string

const (
	Minor    Severity = "minor"
	Major    Severity = "major"
	Critical Severity = "critical"
)

type Incident struct {
	ID, SchoolID, StudentID, ReportedBy, Offence, HearingNotes, Sanction string
	Severity                                                             Severity
	DemeritPoints                                                        int
	RequiresPrincipalApproval                                            bool
	CreatedAt, DecidedAt                                                 time.Time
}

type CounselingNote struct {
	ID, SchoolID, StudentID, CounselorID, Note string
	CreatedAt                                  time.Time
}
