CREATE TABLE ussd_sessions (id text PRIMARY KEY, school_id text NOT NULL REFERENCES tenants(id), phone text NOT NULL, state text NOT NULL, student_id text, last_input text, expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX ussd_sessions_phone_idx ON ussd_sessions(phone, expires_at);
