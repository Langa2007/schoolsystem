package timetable

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type SolverClient interface {
	SolveLessons(context.Context, SolverRequest) (TimetableDraft, error)
}

type SolverRequest struct {
	SchoolID  string
	Requests  []LessonRequest
	Validator Validator
}

type Service struct {
	Solver    SolverClient
	Validator Validator
}

func (s Service) Generate(ctx context.Context, schoolID string, requests []LessonRequest) (TimetableDraft, error) {
	if s.Solver == nil {
		return TimetableDraft{}, errors.New("timetable solver is not configured")
	}
	if schoolID == "" || len(requests) == 0 {
		return TimetableDraft{}, errors.New("school and lesson requests are required")
	}
	draft, err := s.Solver.SolveLessons(ctx, SolverRequest{SchoolID: schoolID, Requests: requests, Validator: s.Validator})
	if err != nil {
		return TimetableDraft{}, fmt.Errorf("solve timetable: %w", err)
	}
	if draft.SchoolID != schoolID {
		return TimetableDraft{}, errors.New("solver returned a different school")
	}
	if err := s.Validator.Validate(requests, draft); err != nil {
		return TimetableDraft{}, fmt.Errorf("solver result rejected: %w", err)
	}
	draft.CreatedAt = time.Now().UTC()
	return draft, nil
}

func (s Service) ValidateManualEdit(requests []LessonRequest, draft TimetableDraft) error {
	return s.Validator.Validate(requests, draft)
}
