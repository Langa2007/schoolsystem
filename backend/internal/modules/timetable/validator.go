package timetable

import (
	"fmt"
	"sort"
)

type Validator struct {
	WorkingDays   []string
	PeriodsPerDay int
	Unavailable   map[string]map[string]map[int]bool
}

func (v Validator) Validate(requests []LessonRequest, draft TimetableDraft) error {
	if len(v.WorkingDays) == 0 || v.PeriodsPerDay < 1 {
		return fmt.Errorf("working days and periods per day are required")
	}
	known := make(map[string]LessonRequest, len(requests))
	for _, request := range requests {
		if request.ID == "" || request.StreamID == "" || request.TeacherID == "" || request.PeriodsPerWeek < 1 || request.Duration < 1 {
			return fmt.Errorf("lesson requests require id, stream, teacher, positive weekly periods, and duration")
		}
		if _, duplicate := known[request.ID]; duplicate {
			return fmt.Errorf("duplicate lesson request %q", request.ID)
		}
		known[request.ID] = request
	}
	occupied := make(map[string]string)
	counts := make(map[string]int)
	for _, placement := range draft.Placements {
		request, ok := known[placement.RequestID]
		if !ok {
			return fmt.Errorf("placement references unknown lesson %q", placement.RequestID)
		}
		duration := placement.Duration
		if duration == 0 {
			duration = request.Duration
		}
		if duration != request.Duration {
			return fmt.Errorf("placement %q duration %d does not match requested duration %d", placement.RequestID, duration, request.Duration)
		}
		if !contains(v.WorkingDays, placement.Day) || placement.Period < 1 || placement.Period+duration-1 > v.PeriodsPerDay {
			return fmt.Errorf("placement %q is outside the school timetable", placement.RequestID)
		}
		for offset := 0; offset < duration; offset++ {
			key := fmt.Sprintf("%s/%d", placement.Day, placement.Period+offset)
			resources := []string{"stream:" + request.StreamID, "teacher:" + request.TeacherID}
			if request.RoomID != "" {
				resources = append(resources, "room:"+request.RoomID)
			}
			for _, resource := range resources {
				if v.Unavailable[resource] != nil && v.Unavailable[resource][placement.Day][placement.Period+offset] {
					return fmt.Errorf("resource %s is unavailable on %s", resource, key)
				}
				if existing, occupiedAlready := occupied[resource+"@"+key]; occupiedAlready {
					return fmt.Errorf("hard clash between %q and %q on %s", existing, placement.RequestID, resource+"@"+key)
				}
				occupied[resource+"@"+key] = placement.RequestID
			}
		}
		counts[placement.RequestID] += duration
	}
	for id := range known {
		if counts[id] == 0 {
			return fmt.Errorf("lesson %q has no placements", id)
		}
	}
	for id, request := range known {
		if counts[id] != request.PeriodsPerWeek {
			return fmt.Errorf("lesson %q has %d periods; expected %d", id, counts[id], request.PeriodsPerWeek)
		}
	}
	return nil
}

func SortPlacements(placements []LessonPlacement) {
	sort.Slice(placements, func(i, j int) bool {
		if placements[i].Day != placements[j].Day {
			return placements[i].Day < placements[j].Day
		}
		return placements[i].Period < placements[j].Period
	})
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
