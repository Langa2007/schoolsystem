-- name: ListStudentLedger :many
SELECT id, student_id, reference_id, description, debit_cents, credit_cents, reversal_of, created_at
FROM fee_ledger_entries WHERE school_id = $1 AND student_id = $2 ORDER BY created_at, id;

-- name: StudentBalanceCents :one
SELECT COALESCE(SUM(debit_cents - credit_cents), 0)::bigint FROM fee_ledger_entries WHERE school_id = $1 AND student_id = $2;

-- name: ListUnmatchedPayments :many
SELECT reference, school_id, admission_number, amount_cents, phone, channel, received_at
FROM mpesa_callbacks WHERE school_id = $1 AND processed_at IS NULL ORDER BY received_at;
