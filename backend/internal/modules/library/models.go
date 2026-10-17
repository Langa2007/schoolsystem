package library

import "time"

type Book struct {
	ID, ISBN, Title   string
	Copies, Available int
	ReplacementCents  int64
}

type Issue struct {
	ID, BookID, StudentID, IssuedBy string
	IssuedAt, DueAt                 time.Time
	ReturnedAt                      *time.Time
	Charged                         bool
}

type ChargePoster interface {
	PostLibraryCharge(studentID, reference string, amountCents int64) error
}
