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
