package domain

import "testing"

func TestFinanceLedgerReversalAndFeeGate(t *testing.T) {
	service, _, parent, notifier := fixture(t)
	bursar := User{ID: "bursar", SchoolID: "school-1", Name: "Bursar", Roles: []Role{RoleBursar}}
	principal := User{ID: "principal", SchoolID: "school-1", Name: "Principal", Roles: []Role{RolePrincipal}}
	if err := service.CreateVoteHead(bursar, VoteHead{ID: "tuition", Name: "Tuition"}); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateInvoice(bursar, Invoice{StudentID: "student-1", TermID: "term-1", VoteHeadID: "tuition", AmountCents: 10000}); err != nil {
		t.Fatal(err)
	}
	if service.BalanceCents("student-1") != 10000 {
		t.Fatalf("expected invoice balance, got %d", service.BalanceCents("student-1"))
	}
	if err := service.SetFeeWithholding(principal, true); err != nil {
		t.Fatal(err)
	}
	if service.CanViewResults("student-1") {
		t.Fatal("expected results to be withheld for a student in arrears")
	}
	entries := service.Ledger("student-1")
	if err := service.ReverseLedgerEntry(parent, entries[0].ID, "unauthorized reversal"); err == nil {
		t.Fatal("expected parent reversal to be rejected")
	}
	if err := service.ReverseLedgerEntry(bursar, entries[0].ID, "approved billing correction"); err != nil {
		t.Fatal(err)
	}
	if service.BalanceCents("student-1") != 0 {
		t.Fatalf("expected reversal to clear balance, got %d", service.BalanceCents("student-1"))
	}
	if len(notifier.Sent) != 0 {
		t.Fatal("finance actions must not send an SMS")
	}
}

func TestMPesaPaymentAllocatesAndQueuesUnknownAdmission(t *testing.T) {
	service, _, _, notifier := fixture(t)
	if err := service.RecordMPesaPayment(Payment{Reference: "QAB123", AdmissionNumber: "A001", AmountCents: 2500, Phone: "+254700000001", Channel: "paybill"}); err != nil {
		t.Fatal(err)
	}
	if service.BalanceCents("student-1") != -2500 || len(notifier.Sent) != 1 {
		t.Fatalf("expected payment allocation and receipt, balance=%d sms=%d", service.BalanceCents("student-1"), len(notifier.Sent))
	}
	if err := service.RecordMPesaPayment(Payment{Reference: "UNKNOWN", AdmissionNumber: "missing", AmountCents: 1000, Channel: "till"}); err != nil {
		t.Fatal(err)
	}
	if len(service.UnmatchedPayments()) != 1 {
		t.Fatal("expected unknown admission to enter unmatched queue")
	}
	if err := service.ResolveUnmatchedPayment(User{ID: "clerk", SchoolID: "school-1", Name: "Clerk", Roles: []Role{RoleAccountsClerk}}, "UNKNOWN", "student-1"); err != nil {
		t.Fatal(err)
	}
	if len(service.UnmatchedPayments()) != 0 || service.BalanceCents("student-1") != -3500 {
		t.Fatal("expected finance staff to allocate the unmatched payment")
	}
}

func TestPrincipalWaiverUnlocksResults(t *testing.T) {
	service, _, _, _ := fixture(t)
	principal := User{ID: "principal", SchoolID: "school-1", Name: "Principal", Roles: []Role{RolePrincipal}}
	bursar := User{ID: "bursar", SchoolID: "school-1", Name: "Bursar", Roles: []Role{RoleBursar}}
	if err := service.CreateVoteHead(bursar, VoteHead{ID: "tuition", Name: "Tuition"}); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateInvoice(bursar, Invoice{StudentID: "student-1", TermID: "term-1", VoteHeadID: "tuition", AmountCents: 1000}); err != nil {
		t.Fatal(err)
	}
	if err := service.SetFeeWithholding(principal, true); err != nil {
		t.Fatal(err)
	}
	if err := service.GrantFeeWaiver(principal, FeeWaiver{StudentID: "student-1", AmountCents: 1000, Reason: "bursary"}); err != nil {
		t.Fatal(err)
	}
	if !service.CanViewResults("student-1") {
		t.Fatal("expected principal waiver to unlock results")
	}
}

// TestSegregationOfDuties verifies that an accounts clerk cannot approve their own
// ledger reversals — only a Principal or Bursar holds that authority. This enforces
// the Phase 3 acceptance criterion: "Segregation of duties prevents accounts clerks
// from approving fee reversals without Principal or Bursar authorization."
func TestSegregationOfDuties(t *testing.T) {
	service, _, _, _ := fixture(t)
	bursar := User{ID: "bursar", SchoolID: "school-1", Name: "Bursar", Roles: []Role{RoleBursar}}
	clerk := User{ID: "clerk", SchoolID: "school-1", Name: "Clerk", Roles: []Role{RoleAccountsClerk}}

	if err := service.CreateVoteHead(bursar, VoteHead{ID: "activity", Name: "Activity"}); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateInvoice(clerk, Invoice{StudentID: "student-1", TermID: "term-1", VoteHeadID: "activity", AmountCents: 5000}); err != nil {
		t.Fatal(err)
	}

	entries := service.Ledger("student-1")
	if len(entries) == 0 {
		t.Fatal("expected at least one ledger entry after invoice")
	}
	entryID := entries[0].ID

	// Accounts clerk must NOT be able to approve a reversal — segregation of duties.
	if err := service.ReverseLedgerEntry(clerk, entryID, "clerk self-reversal"); err == nil {
		t.Fatal("expected accounts clerk to be denied reversal approval — segregation of duties violated")
	}

	// Bursar CAN approve the reversal.
	if err := service.ReverseLedgerEntry(bursar, entryID, "approved by bursar"); err != nil {
		t.Fatalf("bursar should be permitted to reverse: %v", err)
	}
}

// TestDoubleReversalGuard ensures an already-reversed ledger entry cannot be
// reversed a second time, preserving append-only ledger integrity.
func TestDoubleReversalGuard(t *testing.T) {
	service, _, _, _ := fixture(t)
	bursar := User{ID: "bursar", SchoolID: "school-1", Name: "Bursar", Roles: []Role{RoleBursar}}

	if err := service.CreateVoteHead(bursar, VoteHead{ID: "tuition2", Name: "Tuition2"}); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateInvoice(bursar, Invoice{StudentID: "student-1", TermID: "term-1", VoteHeadID: "tuition2", AmountCents: 2000}); err != nil {
		t.Fatal(err)
	}
	entries := service.Ledger("student-1")
	entryID := entries[0].ID

	if err := service.ReverseLedgerEntry(bursar, entryID, "first reversal"); err != nil {
		t.Fatalf("first reversal should succeed: %v", err)
	}
	// Second reversal of the same original entry must be rejected.
	if err := service.ReverseLedgerEntry(bursar, entryID, "second reversal"); err == nil {
		t.Fatal("expected second reversal to be rejected — entry already reversed")
	}
}

// TestDuplicatePaymentReferenceRejected verifies that re-posting the same M-Pesa
// transaction reference is idempotency-guarded to prevent double-crediting.
func TestDuplicatePaymentReferenceRejected(t *testing.T) {
	service, _, _, _ := fixture(t)

	payment := Payment{Reference: "DUPREF01", AdmissionNumber: "A001", AmountCents: 1000, Channel: "paybill"}
	if err := service.RecordMPesaPayment(payment); err != nil {
		t.Fatalf("first recording should succeed: %v", err)
	}
	// Same reference must be rejected.
	if err := service.RecordMPesaPayment(payment); err == nil {
		t.Fatal("expected duplicate payment reference to be rejected")
	}
}

// TestPaymentClearsBalance verifies the happy path: an M-Pesa payment exactly
// matching an invoice zeros the student's outstanding balance and triggers an SMS receipt.
func TestPaymentClearsBalance(t *testing.T) {
	service, _, _, notifier := fixture(t)
	bursar := User{ID: "bursar", SchoolID: "school-1", Name: "Bursar", Roles: []Role{RoleBursar}}

	if err := service.CreateVoteHead(bursar, VoteHead{ID: "boarding", Name: "Boarding"}); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateInvoice(bursar, Invoice{StudentID: "student-1", TermID: "term-1", VoteHeadID: "boarding", AmountCents: 3000}); err != nil {
		t.Fatal(err)
	}
	if service.BalanceCents("student-1") != 3000 {
		t.Fatalf("expected balance 3000 before payment, got %d", service.BalanceCents("student-1"))
	}

	if err := service.RecordMPesaPayment(Payment{Reference: "CLR123", AdmissionNumber: "A001", AmountCents: 3000, Channel: "paybill"}); err != nil {
		t.Fatalf("payment should succeed: %v", err)
	}
	if service.BalanceCents("student-1") != 0 {
		t.Fatalf("expected zero balance after full payment, got %d", service.BalanceCents("student-1"))
	}
	if len(notifier.Sent) != 1 {
		t.Fatalf("expected one payment receipt SMS, got %d", len(notifier.Sent))
	}
}

// TestFeeWithholdingBlocksAndWaiverUnblocks exercises the complete principal fee-gate
// lifecycle: withholding on → results blocked → waiver granted → results visible.
func TestFeeWithholdingBlocksAndWaiverUnblocks(t *testing.T) {
	service, _, _, _ := fixture(t)
	principal := User{ID: "principal", SchoolID: "school-1", Name: "Principal", Roles: []Role{RolePrincipal}}
	bursar := User{ID: "bursar", SchoolID: "school-1", Name: "Bursar", Roles: []Role{RoleBursar}}

	if err := service.CreateVoteHead(bursar, VoteHead{ID: "pta", Name: "PTA"}); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateInvoice(bursar, Invoice{StudentID: "student-1", TermID: "term-1", VoteHeadID: "pta", AmountCents: 500}); err != nil {
		t.Fatal(err)
	}
	// No withholding by default — results visible even in arrears.
	if !service.CanViewResults("student-1") {
		t.Fatal("results should be visible when fee withholding is disabled")
	}

	// Principal enables withholding — now blocked.
	if err := service.SetFeeWithholding(principal, true); err != nil {
		t.Fatal(err)
	}
	if service.CanViewResults("student-1") {
		t.Fatal("results should be blocked after fee withholding is enabled and student has arrears")
	}

	// Non-principal cannot change the withholding flag.
	if err := service.SetFeeWithholding(bursar, false); err == nil {
		t.Fatal("bursar must not be permitted to change fee withholding")
	}

	// Principal grants a waiver — unlocks immediately.
	if err := service.GrantFeeWaiver(principal, FeeWaiver{StudentID: "student-1", AmountCents: 0, Reason: "bursary beneficiary"}); err != nil {
		t.Fatalf("principal waiver should succeed: %v", err)
	}
	if !service.CanViewResults("student-1") {
		t.Fatal("results should be accessible immediately after principal grants a waiver")
	}
}
