from ortools.sat.python import cp_model

from solver.schemas.lesson_input import LessonInput
from solver.schemas.solution_output import Placement, SolutionOutput

def solve(input_data: LessonInput, timeout_seconds: float = 300.0) -> SolutionOutput:
    model = cp_model.CpModel()
    slots = [(day, period) for day in input_data.working_days for period in range(1, input_data.periods_per_day + 1)]
    starts: dict[str, list] = {}
    intervals: dict[str, list] = {}
    for request in input_data.requests:
        starts[request.id] = []
        intervals[request.id] = []
        for index, (day, period) in enumerate(slots):
            if period + request.duration - 1 > input_data.periods_per_day:
                continue
            start = model.NewBoolVar(f"{request.id}_{index}")
            interval = model.NewOptionalFixedSizeIntervalVar(
                period - 1, request.duration, start, f"interval_{request.id}_{index}"
            )
            starts[request.id].append((start, day, period))
            intervals[request.id].append((day, interval, request.stream_id, request.teacher_id, request.room_id))
        # periods_per_week is a count of individual periods; a double period is
        # one placement consuming two periods.
        model.Add(sum(start * request.duration for start, _, _ in starts[request.id]) == request.periods_per_week)

    for resource_key in ("stream", "teacher", "room"):
        resources: dict[str, list] = {}
        for request in input_data.requests:
            resource = getattr(request, f"{resource_key}_id", None)
            if not resource:
                continue
            for day, interval, _, _, _ in intervals[request.id]:
                resources.setdefault(f"{resource_key}:{resource}:{day}", []).append(interval)
        for resource_intervals in resources.values():
            model.AddNoOverlap(resource_intervals)

    solver = cp_model.CpSolver()
    solver.parameters.max_time_in_seconds = timeout_seconds
    solver.parameters.num_search_workers = 8
    status = solver.Solve(model)
    if status not in (cp_model.OPTIMAL, cp_model.FEASIBLE):
        return SolutionOutput(school_id=input_data.school_id, status="infeasible", placements=[], objective=0)
    placements = []
    for request in input_data.requests:
        for start, day, period in starts[request.id]:
            if solver.Value(start):
                placements.append(Placement(request_id=request.id, day=day, period=period, duration=request.duration))
    return SolutionOutput(school_id=input_data.school_id, status="optimal" if status == cp_model.OPTIMAL else "feasible", placements=placements, objective=int(solver.ObjectiveValue()))
