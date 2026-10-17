package sickbay

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Service struct {
	mu       sync.Mutex
	visits   map[string]Visit
	parents  ParentLookup
	notifier Notifier
}

func NewService(parents ParentLookup, notifier Notifier) *Service {
	return &Service{visits: map[string]Visit{}, parents: parents, notifier: notifier}
}

func (s *Service) RecordVisit(visit Visit) error {
	if visit.ID == "" || visit.SchoolID == "" || visit.StudentID == "" || visit.NurseID == "" || strings.TrimSpace(visit.Symptoms) == "" {
		return errors.New("visit identity, nurse, and symptoms are required")
	}
	visit.VisitedAt = time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.visits[visit.ID]; exists {
		return errors.New("visit already exists")
	}
	s.visits[visit.ID] = visit
	if visit.Referral != "" && s.parents != nil && s.notifier != nil {
		for _, phone := range s.parents.ParentPhones(visit.StudentID) {
			if err := s.notifier.SendSMS(phone, fmt.Sprintf("Medical referral recorded for student %s: %s", visit.StudentID, visit.Referral)); err != nil {
				return fmt.Errorf("send medical alert: %w", err)
			}
		}
	}
	return nil
}
