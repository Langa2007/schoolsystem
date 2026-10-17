package jobs

import (
	"context"
	"fmt"

	"elimu-core/backend/internal/modules/timetable"
)

type TimetableSolverJob struct {
	Service timetable.Service
}

func (j TimetableSolverJob) Run(ctx context.Context, schoolID string, requests []timetable.LessonRequest) (timetable.TimetableDraft, error) {
	if j.Service.Solver == nil {
		return timetable.TimetableDraft{}, fmt.Errorf("timetable solver job is not configured")
	}
	return j.Service.Generate(ctx, schoolID, requests)
}
