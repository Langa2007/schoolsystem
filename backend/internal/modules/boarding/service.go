package boarding

import (
	"errors"
	"sync"
	"time"
)

type Service struct {
	mu          sync.RWMutex
	dormitories map[string]Dormitory
	allocations map[string]BedAllocation
	passes      map[string]ExeatPass
	prep        map[string]PrepAttendance
}

func NewService() *Service {
	return &Service{dormitories: map[string]Dormitory{}, allocations: map[string]BedAllocation{}, passes: map[string]ExeatPass{}, prep: map[string]PrepAttendance{}}
}

func (s *Service) AddDormitory(dormitory Dormitory) error {
	if dormitory.ID == "" || dormitory.SchoolID == "" || dormitory.Name == "" || dormitory.Capacity < 1 {
		return errors.New("dormitory identity and positive capacity are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.dormitories[dormitory.ID]; exists {
		return errors.New("dormitory already exists")
	}
	s.dormitories[dormitory.ID] = dormitory
	return nil
}

func (s *Service) AllocateBed(allocation BedAllocation) error {
	if allocation.ID == "" || allocation.DormitoryID == "" || allocation.StudentID == "" {
		return errors.New("allocation identity is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dormitory, ok := s.dormitories[allocation.DormitoryID]
	if !ok {
		return errors.New("dormitory not found")
	}
	occupied := 0
	for _, existing := range s.allocations {
		if existing.DormitoryID == allocation.DormitoryID && existing.ReleasedAt == nil {
			occupied++
		}
		if existing.StudentID == allocation.StudentID && existing.ReleasedAt == nil {
			return errors.New("student already has an active bed")
		}
	}
	if occupied >= dormitory.Capacity {
		return errors.New("dormitory is full")
	}
	allocatedAt := time.Now().UTC()
	allocation.AllocatedAt = &allocatedAt
	s.allocations[allocation.ID] = allocation
	return nil
}

func (s *Service) IssueExeat(pass ExeatPass) error {
	if pass.ID == "" || pass.StudentID == "" || pass.ApprovedBy == "" || pass.Departure.IsZero() || pass.ExpectedReturn.Before(pass.Departure) {
		return errors.New("valid approved exeat details are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.passes[pass.ID] = pass
	return nil
}

func (s *Service) ReturnExeat(id string, returnedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pass, ok := s.passes[id]
	if !ok {
		return errors.New("exeat pass not found")
	}
	if returnedAt.IsZero() {
		return errors.New("return time is required")
	}
	pass.ReturnedAt = &returnedAt
	s.passes[id] = pass
	return nil
}

func (s *Service) RecordPrep(attendance PrepAttendance) error {
	if attendance.Date == "" || attendance.DormitoryID == "" {
		return errors.New("date and dormitory are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prep[attendance.Date+"|"+attendance.DormitoryID] = attendance
	return nil
}
