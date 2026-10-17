package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type VoteHead struct {
	ID, SchoolID, Name string
	Active             bool
}

type Invoice struct {
	ID, StudentID, TermID, VoteHeadID string
	AmountCents                       int64
	CreatedAt                         time.Time
}

type LedgerEntry struct {
	ID, StudentID, ReferenceID, Description string
	DebitCents, CreditCents                 int64
	CreatedAt                               time.Time
	ReversalOf                              string
}

type Payment struct {
	Reference, AdmissionNumber, StudentID, Phone, Channel string
	AmountCents                                           int64
	ReceivedAt                                            time.Time
	Allocated                                             bool
}

type FeeWaiver struct {
	ID, StudentID, Reason, ApprovedBy string
	AmountCents                       int64
	CreatedAt                         time.Time
}

func financeActor(actor User) bool {
	return hasRole(actor, RolePrincipal) || hasRole(actor, RoleBursar) || hasRole(actor, RoleAccountsClerk)
}

func (s *Service) CreateVoteHead(actor User, head VoteHead) error {
	if !financeActor(actor) || hasRole(actor, RoleAccountsClerk) {
		return errors.New("principal or bursar required")
	}
	if strings.TrimSpace(head.ID) == "" || strings.TrimSpace(head.Name) == "" {
		return errors.New("vote head id and name are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.voteHeads[head.ID]; exists {
		return errors.New("vote head already exists")
	}
	head.SchoolID, head.Active = s.settings.ID, true
	s.voteHeads[head.ID] = head
	s.audit(actor, "finance.vote_head_created", "vote_head", head.ID, head.Name)
	return nil
}

func (s *Service) CreateInvoice(actor User, invoice Invoice) error {
	if !financeActor(actor) {
		return errors.New("finance role required")
	}
	if invoice.StudentID == "" || invoice.TermID == "" || invoice.VoteHeadID == "" || invoice.AmountCents <= 0 {
		return errors.New("student, term, vote head, and positive amount are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.students[invoice.StudentID]; !ok {
		return errors.New("student not found")
	}
	if head, ok := s.voteHeads[invoice.VoteHeadID]; !ok || !head.Active {
		return errors.New("active vote head not found")
	}
	invoice.ID, invoice.CreatedAt = randomID(), time.Now().UTC()
	s.invoices[invoice.ID] = invoice
	s.ledger = append(s.ledger, LedgerEntry{ID: randomID(), StudentID: invoice.StudentID, ReferenceID: invoice.ID, Description: "invoice", DebitCents: invoice.AmountCents, CreatedAt: invoice.CreatedAt})
	s.audit(actor, "finance.invoice_created", "invoice", invoice.ID, fmt.Sprintf("amount_cents=%d", invoice.AmountCents))
	return nil
}

func (s *Service) Ledger(studentID string) []LedgerEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]LedgerEntry, 0)
	for _, entry := range s.ledger {
		if entry.StudentID == studentID {
			result = append(result, entry)
		}
	}
	return result
}

func (s *Service) BalanceCents(studentID string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var balance int64
	for _, entry := range s.ledger {
		if entry.StudentID == studentID {
			balance += entry.DebitCents - entry.CreditCents
		}
	}
	return balance
}

func (s *Service) ReverseLedgerEntry(actor User, entryID, reason string) error {
	if !hasRole(actor, RolePrincipal) && !hasRole(actor, RoleBursar) {
		return errors.New("principal or bursar authorization required")
	}
	if strings.TrimSpace(reason) == "" {
		return errors.New("reversal reason is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, entry := range s.ledger {
		if entry.ID != entryID {
			continue
		}
		for _, existing := range s.ledger {
			if existing.ReversalOf == entryID {
				return errors.New("ledger entry is already reversed")
			}
		}
		reversal := LedgerEntry{ID: randomID(), StudentID: entry.StudentID, ReferenceID: entry.ReferenceID, Description: reason, DebitCents: entry.CreditCents, CreditCents: entry.DebitCents, CreatedAt: time.Now().UTC(), ReversalOf: entry.ID}
		s.ledger = append(s.ledger, reversal)
		s.audit(actor, "finance.ledger_reversed", "ledger_entry", entry.ID, reason)
		return nil
	}
	return errors.New("ledger entry not found")
}

func (s *Service) SetFeeWithholding(actor User, enabled bool) error {
	if !hasRole(actor, RolePrincipal) {
		return errors.New("only a principal can change fee withholding")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings.FeeWithholdingEnabled = enabled
	s.audit(actor, "finance.fee_withholding_changed", "school", s.settings.ID, fmt.Sprintf("enabled=%t", enabled))
	return nil
}

func (s *Service) GrantFeeWaiver(actor User, waiver FeeWaiver) error {
	if !hasRole(actor, RolePrincipal) {
		return errors.New("only a principal can grant fee waivers")
	}
	if waiver.StudentID == "" || waiver.AmountCents < 0 || strings.TrimSpace(waiver.Reason) == "" {
		return errors.New("student, non-negative amount, and reason are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.students[waiver.StudentID]; !ok {
		return errors.New("student not found")
	}
	waiver.ID, waiver.ApprovedBy, waiver.CreatedAt = randomID(), actor.ID, time.Now().UTC()
	s.waivers[waiver.StudentID] = waiver
	s.ledger = append(s.ledger, LedgerEntry{ID: randomID(), StudentID: waiver.StudentID, ReferenceID: waiver.ID, Description: "approved fee waiver", CreditCents: waiver.AmountCents, CreatedAt: waiver.CreatedAt})
	s.audit(actor, "finance.fee_waiver_granted", "fee_waiver", waiver.ID, waiver.Reason)
	return nil
}

func (s *Service) CanViewResults(studentID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.settings.FeeWithholdingEnabled || s.waivers[studentID].ID != "" {
		return true
	}
	var balance int64
	for _, entry := range s.ledger {
		if entry.StudentID == studentID {
			balance += entry.DebitCents - entry.CreditCents
		}
	}
	return balance <= 0
}

func (s *Service) RecordMPesaPayment(payment Payment) error {
	if payment.Reference == "" || payment.AdmissionNumber == "" || payment.AmountCents <= 0 {
		return errors.New("reference, admission number, and positive amount are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.payments[payment.Reference]; exists {
		return errors.New("payment reference already processed")
	}
	if _, exists := s.unmatched[payment.Reference]; exists {
		return errors.New("payment reference already received")
	}
	payment.ReceivedAt = time.Now().UTC()
	for id, student := range s.students {
		if student.AdmissionNumber == payment.AdmissionNumber {
			payment.StudentID, payment.Allocated = id, true
			break
		}
	}
	if !payment.Allocated {
		s.unmatched[payment.Reference] = payment
		return nil
	}
	s.payments[payment.Reference] = payment
	s.ledger = append(s.ledger, LedgerEntry{ID: randomID(), StudentID: payment.StudentID, ReferenceID: payment.Reference, Description: "M-Pesa payment", CreditCents: payment.AmountCents, CreatedAt: payment.ReceivedAt})
	s.audit(User{ID: "system", SchoolID: s.settings.ID}, "finance.mpesa_payment_received", "payment", payment.Reference, payment.AdmissionNumber)
	student := s.students[payment.StudentID]
	for _, parentID := range student.ParentIDs {
		if parent, ok := s.users[parentID]; ok && parent.Phone != "" {
			if err := s.notifier.SendSMS(parent.Phone, fmt.Sprintf("Payment %s received for %s.", payment.Reference, student.Name)); err != nil {
				return fmt.Errorf("send payment receipt: %w", err)
			}
		}
	}
	return nil
}

func (s *Service) UnmatchedPayments() []Payment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Payment, 0, len(s.unmatched))
	for _, payment := range s.unmatched {
		result = append(result, payment)
	}
	return result
}

func (s *Service) PostLibraryCharge(studentID, reference string, amountCents int64) error {
	if studentID == "" || reference == "" || amountCents <= 0 {
		return errors.New("student, reference, and positive charge are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.students[studentID]; !ok {
		return errors.New("student not found")
	}
	s.ledger = append(s.ledger, LedgerEntry{
		ID: randomID(), StudentID: studentID, ReferenceID: reference,
		Description: "library replacement charge", DebitCents: amountCents,
		CreatedAt: time.Now().UTC(),
	})
	return nil
}

func (s *Service) ResolveUnmatchedPayment(actor User, reference, studentID string) error {
	if !financeActor(actor) {
		return errors.New("finance role required")
	}
	if reference == "" || studentID == "" {
		return errors.New("reference and student are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	payment, ok := s.unmatched[reference]
	if !ok {
		return errors.New("unmatched payment not found")
	}
	student, ok := s.students[studentID]
	if !ok {
		return errors.New("student not found")
	}
	payment.StudentID, payment.Allocated = studentID, true
	s.payments[reference] = payment
	delete(s.unmatched, reference)
	s.ledger = append(s.ledger, LedgerEntry{ID: randomID(), StudentID: studentID, ReferenceID: reference, Description: "allocated M-Pesa payment", CreditCents: payment.AmountCents, CreatedAt: payment.ReceivedAt})
	s.audit(actor, "finance.unmatched_payment_resolved", "payment", reference, studentID)
	for _, parentID := range student.ParentIDs {
		if parent, ok := s.users[parentID]; ok && parent.Phone != "" {
			if err := s.notifier.SendSMS(parent.Phone, fmt.Sprintf("Payment %s received for %s.", reference, student.Name)); err != nil {
				return fmt.Errorf("send payment receipt: %w", err)
			}
		}
	}
	return nil
}
