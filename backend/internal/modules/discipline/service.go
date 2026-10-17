package discipline

import (
	"errors"
	"strings"
	"sync"
	"time"
)

type Service struct {
	mu         sync.RWMutex
	incidents  map[string]Incident
	notes      map[string]CounselingNote
	counselors map[string]bool
}

func NewService() *Service {
	return &Service{incidents: make(map[string]Incident), notes: make(map[string]CounselingNote), counselors: make(map[string]bool)}
}

func (s *Service) RegisterCounselor(userID string) error {
	if strings.TrimSpace(userID) == "" {
		return errors.New("counselor id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counselors[userID] = true
	return nil
}

func (s *Service) RecordIncident(incident Incident) error {
	if incident.ID == "" || incident.SchoolID == "" || incident.StudentID == "" || incident.ReportedBy == "" || strings.TrimSpace(incident.Offence) == "" {
		return errors.New("incident identity, reporter, and offence are required")
	}
	if incident.DemeritPoints < 0 {
		return errors.New("demerit points cannot be negative")
	}
	incident.CreatedAt = time.Now().UTC()
	incident.RequiresPrincipalApproval = incident.Severity == Major || incident.Severity == Critical
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.incidents[incident.ID]; exists {
		return errors.New("incident already exists")
	}
	s.incidents[incident.ID] = incident
	return nil
}

func (s *Service) ApproveIncident(id, principalID, sanction string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[id]
	if !ok {
		return errors.New("incident not found")
	}
	if incident.RequiresPrincipalApproval && principalID == "" {
		return errors.New("principal approval is required")
	}
	if strings.TrimSpace(sanction) == "" {
		return errors.New("sanction is required")
	}
	incident.Sanction, incident.DecidedAt = sanction, time.Now().UTC()
	s.incidents[id] = incident
	return nil
}

func (s *Service) AddCounselingNote(note CounselingNote) error {
	if note.ID == "" || note.SchoolID == "" || note.StudentID == "" || note.CounselorID == "" || strings.TrimSpace(note.Note) == "" {
		return errors.New("complete counseling note is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.counselors[note.CounselorID] {
		return errors.New("counselor access denied")
	}
	note.CreatedAt = time.Now().UTC()
	s.notes[note.ID] = note
	return nil
}

func (s *Service) CounselingNotes(requesterID, schoolID, studentID string) ([]CounselingNote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.counselors[requesterID] {
		return nil, errors.New("counselor access denied")
	}
	result := make([]CounselingNote, 0)
	for _, note := range s.notes {
		if note.SchoolID == schoolID && note.StudentID == studentID {
			result = append(result, note)
		}
	}
	return result, nil
}
