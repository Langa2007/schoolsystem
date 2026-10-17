package sickbay

import "time"

type Visit struct {
	ID, SchoolID, StudentID, NurseID, Symptoms, Treatment, Referral string
	ParentAlert                                                     bool
	VisitedAt                                                       time.Time
}

type Notifier interface {
	SendSMS(to, message string) error
}

type ParentLookup interface {
	ParentPhones(studentID string) []string
}
