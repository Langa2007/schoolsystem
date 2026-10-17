package library

import (
	"testing"
	"time"
)

type charges struct {
	studentID, reference string
	amount               int64
}

func (c *charges) PostLibraryCharge(studentID, reference string, amount int64) error {
	c.studentID, c.reference, c.amount = studentID, reference, amount
	return nil
}

func TestLostBookPostsReplacementCharge(t *testing.T) {
	poster := &charges{}
	service := NewService(poster)
	if err := service.AddBook(Book{ID: "book-1", Title: "Mathematics", Copies: 1, Available: 1, ReplacementCents: 250000}); err != nil {
		t.Fatal(err)
	}
	if err := service.Issue(Issue{ID: "issue-1", BookID: "book-1", StudentID: "student-1", IssuedBy: "librarian", DueAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := service.Return("issue-1", true); err != nil {
		t.Fatal(err)
	}
	if poster.studentID != "student-1" || poster.amount != 250000 {
		t.Fatalf("expected replacement charge, got %#v", poster)
	}
}
