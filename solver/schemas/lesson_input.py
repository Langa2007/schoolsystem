from pydantic import BaseModel, Field

class LessonRequest(BaseModel):
    id: str
    stream_id: str
    subject_id: str
    teacher_id: str
    room_id: str | None = None
    periods_per_week: int = Field(ge=1, le=20)
    duration: int = Field(default=1, ge=1, le=3)

class LessonInput(BaseModel):
    school_id: str
    working_days: list[str] = Field(min_length=1)
    periods_per_day: int = Field(ge=1, le=20)
    requests: list[LessonRequest] = Field(min_length=1)
