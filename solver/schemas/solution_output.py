from pydantic import BaseModel

class Placement(BaseModel):
    request_id: str
    day: str
    period: int
    duration: int

class SolutionOutput(BaseModel):
    school_id: str
    status: str
    placements: list[Placement]
    objective: int
