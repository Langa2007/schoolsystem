package solver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"elimu-core/backend/internal/modules/timetable"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

type request struct {
	SchoolID      string          `json:"school_id"`
	WorkingDays   []string        `json:"working_days"`
	PeriodsPerDay int             `json:"periods_per_day"`
	Requests      []lessonRequest `json:"requests"`
}

type lessonRequest struct {
	ID, StreamID, SubjectID, TeacherID, RoomID string
	PeriodsPerWeek, Duration                   int
}

type response struct {
	SchoolID   string      `json:"school_id"`
	Status     string      `json:"status"`
	Placements []placement `json:"placements"`
}

type placement struct {
	RequestID        string `json:"request_id"`
	Day              string `json:"day"`
	Period, Duration int
}

func (c Client) SolveLessons(ctx context.Context, input timetable.SolverRequest) (timetable.TimetableDraft, error) {
	if strings.TrimSpace(c.BaseURL) == "" {
		return timetable.TimetableDraft{}, errors.New("solver base URL is required")
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	payload := request{SchoolID: input.SchoolID, WorkingDays: input.Validator.WorkingDays, PeriodsPerDay: input.Validator.PeriodsPerDay}
	for _, item := range input.Requests {
		payload.Requests = append(payload.Requests, lessonRequest{ID: item.ID, StreamID: item.StreamID, SubjectID: item.SubjectID, TeacherID: item.TeacherID, RoomID: item.RoomID, PeriodsPerWeek: item.PeriodsPerWeek, Duration: item.Duration})
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return timetable.TimetableDraft{}, fmt.Errorf("encode solver request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/v1/solve/lessons", bytes.NewReader(body))
	if err != nil {
		return timetable.TimetableDraft{}, fmt.Errorf("create solver request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return timetable.TimetableDraft{}, fmt.Errorf("call solver: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return timetable.TimetableDraft{}, fmt.Errorf("solver returned HTTP %d", res.StatusCode)
	}
	var decoded response
	if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
		return timetable.TimetableDraft{}, fmt.Errorf("decode solver response: %w", err)
	}
	draft := timetable.TimetableDraft{SchoolID: decoded.SchoolID}
	for _, item := range decoded.Placements {
		draft.Placements = append(draft.Placements, timetable.LessonPlacement{RequestID: item.RequestID, Slot: timetable.Slot{Day: item.Day, Period: item.Period}, Duration: item.Duration})
	}
	return draft, nil
}
