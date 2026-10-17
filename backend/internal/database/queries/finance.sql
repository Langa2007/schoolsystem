-- name: ListStudentLedger :many
SELECT id, student_id, reference_id, description, debit_cents, credit_cents, reversal_of, created_at
FROM fee_ledger_entries WHERE school_id = $1 AND student_id = $2 ORDER BY created_at, id;

-- name: StudentBalanceCents :one
SELECT COALESCE(SUM(debit_cents - credit_cents), 0)::bigint FROM fee_ledger_entries WHERE school_id = $1 AND student_id = $2;

-- name: ListUnmatchedPayments :many
SELECT reference, school_id, admission_number, amount_cents, phone, channel, received_at
FROM mpesa_callbacks WHERE school_id = $1 AND processed_at IS NULL ORDER BY received_at;

-- name: InsertLedgerEntry :exec
INSERT INTO fee_ledger_entries (id, school_id, student_id, reference_id, description, debit_cents, credit_cents, reversal_of)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: LedgerEntryByID :one
SELECT id, school_id, student_id, reference_id, description, debit_cents, credit_cents, reversal_of, created_at
FROM fee_ledger_entries WHERE school_id = $1 AND id = $2;

-- name: LedgerEntryReversalExists :one
SELECT EXISTS(SELECT 1 FROM fee_ledger_entries WHERE school_id = $1 AND reversal_of = $2)::boolean;

-- name: InsertVoteHead :exec
INSERT INTO vote_heads (id, school_id, name, active) VALUES ($1, $2, $3, true);

-- name: GetVoteHead :one
SELECT id, school_id, name, active FROM vote_heads WHERE school_id = $1 AND id = $2;

-- name: ListVoteHeads :many
SELECT id, school_id, name, active FROM vote_heads WHERE school_id = $1 AND active = true ORDER BY name;

-- name: InsertInvoice :exec
INSERT INTO invoices (id, school_id, student_id, term_id, vote_head_id, amount_cents)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListStudentInvoices :many
SELECT id, student_id, term_id, vote_head_id, amount_cents, created_at
FROM invoices WHERE school_id = $1 AND student_id = $2 ORDER BY created_at;

-- name: InsertFeeWaiver :exec
INSERT INTO fee_result_waivers (id, school_id, student_id, amount_cents, reason, approved_by)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: StudentHasFeeWaiver :one
SELECT EXISTS(SELECT 1 FROM fee_result_waivers WHERE school_id = $1 AND student_id = $2)::boolean;

-- name: InsertMPesaCallback :exec
INSERT INTO mpesa_callbacks (reference, school_id, admission_number, amount_cents, phone, channel)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: MarkMPesaCallbackProcessed :exec
UPDATE mpesa_callbacks SET processed_at = now() WHERE reference = $1;

-- name: InsertUnmatchedPayment :exec
INSERT INTO unmatched_payments (reference, school_id) VALUES ($1, $2);

-- name: ResolveUnmatchedPayment :exec
UPDATE unmatched_payments SET resolved_by = $1, resolution_note = $2, resolved_at = now()
WHERE reference = $3;
