CREATE TABLE exam_sheets (
    id text PRIMARY KEY,
    school_id text NOT NULL REFERENCES tenants(id),
    class_stream text NOT NULL,
    subject text NOT NULL,
    teacher_id text NOT NULL REFERENCES users(id),
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'verified', 'published')),
    dos_approved boolean NOT NULL DEFAULT false,
    principal_approved boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE exam_marks (
    exam_sheet_id text NOT NULL REFERENCES exam_sheets(id),
    student_id text NOT NULL REFERENCES students(id),
    score integer NOT NULL CHECK (score BETWEEN 0 AND 100),
    PRIMARY KEY (exam_sheet_id, student_id)
);

CREATE TABLE correction_tickets (
    id text PRIMARY KEY,
    exam_sheet_id text NOT NULL REFERENCES exam_sheets(id),
    student_id text NOT NULL REFERENCES students(id),
    previous_score integer NOT NULL CHECK (previous_score BETWEEN 0 AND 100),
    requested_score integer NOT NULL CHECK (requested_score BETWEEN 0 AND 100),
    justification text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    officer_id text REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    decided_at timestamptz
);

CREATE OR REPLACE FUNCTION prevent_submitted_mark_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM exam_sheets
        WHERE id = OLD.exam_sheet_id AND status <> 'draft'
    ) THEN
        RAISE EXCEPTION 'submitted exam marks are immutable';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER exam_marks_immutable
BEFORE UPDATE OR DELETE ON exam_marks
FOR EACH ROW EXECUTE FUNCTION prevent_submitted_mark_mutation();
