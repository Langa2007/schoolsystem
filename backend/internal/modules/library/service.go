package library

import (
	"errors"
	"sync"
	"time"
)

type Service struct {
	mu      sync.Mutex
	books   map[string]Book
	issues  map[string]Issue
	charges ChargePoster
}

func NewService(charges ChargePoster) *Service {
	return &Service{books: map[string]Book{}, issues: map[string]Issue{}, charges: charges}
}

func (s *Service) AddBook(book Book) error {
	if book.ID == "" || book.Title == "" || book.Copies < 1 || book.Available < 0 || book.Available > book.Copies || book.ReplacementCents < 0 {
		return errors.New("invalid book record")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.books[book.ID]; exists {
		return errors.New("book already exists")
	}
	s.books[book.ID] = book
	return nil
}

func (s *Service) Issue(issue Issue) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	book, ok := s.books[issue.BookID]
	if !ok {
		return errors.New("book not found")
	}
	if book.Available < 1 {
		return errors.New("no available copy")
	}
	if issue.ID == "" || issue.StudentID == "" || issue.IssuedBy == "" || issue.DueAt.IsZero() {
		return errors.New("complete issue details are required")
	}
	if _, exists := s.issues[issue.ID]; exists {
		return errors.New("issue already exists")
	}
	book.Available--
	issue.IssuedAt = time.Now().UTC()
	s.books[book.ID], s.issues[issue.ID] = book, issue
	return nil
}

func (s *Service) Return(issueID string, lost bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	issue, ok := s.issues[issueID]
	if !ok {
		return errors.New("issue not found")
	}
	if issue.ReturnedAt != nil {
		return errors.New("book already returned")
	}
	book, exists := s.books[issue.BookID]
	if !exists {
		return errors.New("book not found")
	}
	if lost && (s.charges == nil || book.ReplacementCents < 1) {
		return errors.New("lost book requires a configured finance poster and positive replacement charge")
	}
	now := time.Now().UTC()
	issue.ReturnedAt, issue.Charged = &now, lost
	if !lost {
		book.Available++
	} else if s.charges != nil {
		if err := s.charges.PostLibraryCharge(issue.StudentID, "library:"+issue.ID, book.ReplacementCents); err != nil {
			return err
		}
	}
	s.books[book.ID], s.issues[issueID] = book, issue
	return nil
}
