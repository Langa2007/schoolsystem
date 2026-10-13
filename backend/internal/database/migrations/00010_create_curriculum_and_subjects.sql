CREATE TABLE cbc_enrollments (
    student_id text PRIMARY KEY REFERENCES students(id),
    grade smallint NOT NULL CHECK (grade BETWEEN 10 AND 12),
    pathway text NOT NULL CHECK (pathway IN ('stem', 'social_sciences', 'arts_and_sports_science')),
    electives text[] NOT NULL DEFAULT '{}'
);

CREATE TABLE competency_assessments (
    id text PRIMARY KEY,
    student_id text NOT NULL REFERENCES students(id),
    subject text NOT NULL,
    term_id text NOT NULL REFERENCES academic_terms(id),
    level text NOT NULL CHECK (level IN ('EE', 'ME', 'AE', 'BE')),
    portfolio text,
    project_note text,
    recorded_by text NOT NULL REFERENCES users(id),
    recorded_at timestamptz NOT NULL DEFAULT now()
);
