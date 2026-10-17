from fastapi import FastAPI, HTTPException

from solver.schemas.lesson_input import LessonInput
from solver.schemas.solution_output import SolutionOutput
from solver.solvers.lesson_timetable_solver import solve

app = FastAPI(title="ElimuCore Scheduling Solver", version="1.0.0")

@app.get("/healthz")
def health() -> dict[str, str]:
    return {"status": "ok"}

@app.post("/v1/solve/lessons", response_model=SolutionOutput)
def solve_lessons(payload: LessonInput) -> SolutionOutput:
    result = solve(payload)
    if result.status == "infeasible":
        raise HTTPException(status_code=422, detail="timetable constraints are infeasible")
    return result
