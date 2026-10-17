from solver.schemas.lesson_input import LessonInput, LessonRequest
from solver.solvers.lesson_timetable_solver import solve

def test_solver_avoids_teacher_clashes():
    payload = LessonInput(
        school_id="school-1",
        working_days=["mon", "tue"],
        periods_per_day=4,
        requests=[
            LessonRequest(id="a", stream_id="stream-a", subject_id="math", teacher_id="teacher-1", periods_per_week=2),
            LessonRequest(id="b", stream_id="stream-b", subject_id="eng", teacher_id="teacher-1", periods_per_week=2),
        ],
    )
    result = solve(payload, timeout_seconds=5)
    assert result.status in {"optimal", "feasible"}
    occupied = {(item.day, item.period) for item in result.placements}
    assert len(occupied) == len(result.placements)
