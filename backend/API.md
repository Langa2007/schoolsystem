# Backend integration API

The current API is a development/integration server. It uses an in-memory domain service; records reset when the process restarts. PostgreSQL migrations are embedded and can be applied when `DATABASE_URL` is supplied, but domain reads and writes are not yet backed by PostgreSQL. Production startup therefore fails closed.

## Local startup

Set `BOOTSTRAP_ADMIN_PASSWORD` to a password with at least eight characters, including a letter and a number. `BOOTSTRAP_ADMIN_ID` defaults to `admin`. In development, if `SESSION_SECRET` is omitted, the server creates an ephemeral secret and all sessions expire when it restarts.

```sh
cd backend
BOOTSTRAP_ADMIN_PASSWORD='change-me-123' go run ./cmd/server
```

To create the schema and the configured school row, run `PGUSER=<postgres-role> PGPASSWORD=<postgres-password> go run ./cmd/dbsetup` from `backend/`. The setup command reads `DATABASE_URL` and optional `SCHOOL_*` settings from the repository `.env`, accepts JDBC-prefixed PostgreSQL URLs, applies migrations once using `schema_migrations`, and inserts standard vote heads idempotently. It does not create sample students or staff.

Login with `POST /api/v1/auth/login` and `{ "identifier": "admin", "password": "change-me-123" }`. Send the returned token as `Authorization: Bearer <access_token>`. Provisioned accounts return `must_change_password: true`; change it through `POST /api/v1/auth/password` before continuing. `GET /api/v1/auth/me` returns the current user and roles. `GET /healthz` checks process liveness; `GET /readyz` reports whether the process is ready and explicitly reports `persistence: in_memory`.

## Implemented HTTP routes

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/api/v1/auth/login` | Authenticate by user ID, phone, or email and issue a bearer token |
| GET | `/api/v1/settings` | School settings |
| GET | `/api/v1/auth/me` | Return the authenticated account and current roles |
| POST | `/api/v1/auth/password` | Change the current user's first-login or current password |
| GET | `/api/v1/students` | List records visible to a linked parent, assigned class teacher, or school administrator |
| POST | `/api/v1/users` | Principal/registrar provisions an account and initial password |
| POST | `/api/v1/students` | Register a student |
| POST | `/api/v1/staff` | Register a staff profile |
| GET | `/api/v1/terms` | List academic terms |
| POST | `/api/v1/terms` | Create an academic term |
| GET | `/api/v1/streams` | List class streams |
| POST | `/api/v1/streams` | Create a class stream |
| POST | `/api/v1/attendance/prefect-count` | Submit a prefect headcount |
| POST | `/api/v1/attendance/morning-roll` | Post the official morning roll |
| POST | `/api/v1/attendance/lessons` | Record lesson delivery |
| POST | `/api/v1/attendance/reconcile` | Reconcile attendance gaps |
| GET | `/api/v1/attendance/summary?date=YYYY-MM-DD&stream_id=...` | Read a class attendance summary |
| POST | `/api/v1/leave-requests` | Submit a leave request |
| POST | `/api/v1/leave-requests/{id}/decision` | Approve or reject a leave request |
| POST | `/api/v1/cbc/enrollments` | Enroll a student in a CBE pathway |
| POST | `/api/v1/cbc/assessments` | Record a competency assessment |
| POST | `/api/v1/exams/sheets` | Create a draft mark sheet |
| PUT | `/api/v1/exams/sheets/{id}/marks/{student_id}` | Set a draft mark |
| POST | `/api/v1/exams/sheets/{id}/submit` | Lock and submit a mark sheet |
| GET | `/api/v1/exams/sheets/{id}/verification` | Get an admission-sorted verification list |
| POST | `/api/v1/finance/vote-heads` | Create a vote head |
| POST | `/api/v1/finance/invoices` | Create an invoice |
| GET | `/api/v1/finance/ledger/{student_id}` | Read an authorized student's ledger |
| POST | `/api/v1/finance/ledger/reverse` | Reverse a ledger entry with role checks |
| POST | `/api/v1/finance/fee-withholding` | Set the principal fee-results gate |
| POST | `/api/v1/finance/waivers` | Grant an individual fee waiver |
| GET | `/api/v1/finance/results-access/{student_id}` | Check report-card access |
| POST | `/api/v1/finance/mpesa/callback?token=...` | Allocate a C2B callback after callback-token validation |
| POST | `/api/v1/finance/mpesa/stk-push` | Initiate STK Push when Daraja is configured |
| GET | `/api/v1/finance/mpesa/unmatched` | List unallocated payments |
| POST | `/api/v1/finance/mpesa/unmatched/resolve` | Assign an unmatched payment to a student |

Finance endpoints are registered in `cmd/server/main.go`. Authenticated finance routes require a valid token. The C2B callback requires `?token=<MPESA_CALLBACK_TOKEN>`; configure a long random token in Daraja's confirmation URL. STK Push requires all `DARAJA_*` settings. External SMS and production persistence still need adapters/repositories.

Set `CORS_ALLOWED_ORIGINS` to a comma-separated list of exact frontend origins. The development default includes localhost ports 3000 and 5173.

## Not yet integration-ready

Phase 4 operations and the Phase 5 sync/USSD/reporting modules do not yet have their full HTTP route sets. PostgreSQL migrations exist, but business services are in-memory and production mode is intentionally disabled until repository wiring is complete. The web frontend should treat the route table above as the current contract and use mocks for workflows that are not listed.
