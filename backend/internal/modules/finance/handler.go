package finance

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"elimu-core/backend/internal/domain"
	"elimu-core/backend/internal/pkg/daraja"
	"elimu-core/backend/internal/pkg/identity"
	"elimu-core/backend/internal/pkg/response"
)

// ContextWithActor returns a shallow copy of r with the actor embedded in its context.
// The auth middleware calls this before routing to any finance handler.
func ContextWithActor(r *http.Request, actor domain.User) *http.Request {
	return r.WithContext(identity.WithActor(r.Context(), actor))
}

func actorFromContext(r *http.Request) (domain.User, bool) {
	return identity.Actor(r.Context())
}

// Handler exposes all Phase 3 finance HTTP endpoints.
// DarajaClient is optional: when nil, the STK-push endpoint responds with 503
// so the portal degrades gracefully until production credentials are deployed.
type Handler struct {
	Service            *domain.Service
	DarajaClient       daraja.Client
	DarajaShortCode    string
	MPesaCallbackToken string
}

// ---------- Vote Heads ----------

type createVoteHeadRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateVoteHead handles POST /api/v1/finance/vote-heads.
// Only Principal or Bursar may create vote heads; accounts clerks are excluded.
func (h Handler) CreateVoteHead(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req createVoteHeadRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	req.ID = strings.TrimSpace(req.ID)
	req.Name = strings.TrimSpace(req.Name)
	if err := h.Service.CreateVoteHead(actor, domain.VoteHead{ID: req.ID, Name: req.Name}); err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	_ = response.JSON(w, http.StatusCreated, map[string]string{"id": req.ID})
}

// ---------- Invoices ----------

type createInvoiceRequest struct {
	StudentID   string `json:"student_id"`
	TermID      string `json:"term_id"`
	VoteHeadID  string `json:"vote_head_id"`
	AmountCents int64  `json:"amount_cents"`
}

// CreateInvoice handles POST /api/v1/finance/invoices.
func (h Handler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req createInvoiceRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	inv := domain.Invoice{
		StudentID:   strings.TrimSpace(req.StudentID),
		TermID:      strings.TrimSpace(req.TermID),
		VoteHeadID:  strings.TrimSpace(req.VoteHeadID),
		AmountCents: req.AmountCents,
	}
	if err := h.Service.CreateInvoice(actor, inv); err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	_ = response.JSON(w, http.StatusCreated, map[string]string{"status": "invoiced"})
}

// ---------- Ledger ----------

// StudentLedger handles GET /api/v1/finance/ledger/{student_id}.
// Returns the full ledger entry list and current balance for a student.
func (h Handler) StudentLedger(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	studentID := r.PathValue("student_id")
	if strings.TrimSpace(studentID) == "" {
		response.Error(w, http.StatusBadRequest, "student_id is required")
		return
	}
	if !h.Service.CanAccessStudent(actor, studentID) {
		response.Error(w, http.StatusForbidden, "student ledger access denied")
		return
	}
	entries := h.Service.Ledger(studentID)
	balance := h.Service.BalanceCents(studentID)
	_ = response.JSON(w, http.StatusOK, map[string]any{
		"student_id":    studentID,
		"entries":       entries,
		"balance_cents": balance,
	})
}

// ---------- Ledger Reversal (Segregation of Duties) ----------

type reversalRequest struct {
	EntryID string `json:"entry_id"`
	Reason  string `json:"reason"`
}

// ReverseLedgerEntry handles POST /api/v1/finance/ledger/reverse.
// Segregation of duties: only Principal or Bursar may approve reversals.
// Accounts clerks cannot self-approve fee corrections; the domain enforces this.
func (h Handler) ReverseLedgerEntry(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req reversalRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	req.EntryID = strings.TrimSpace(req.EntryID)
	req.Reason = strings.TrimSpace(req.Reason)
	if err := h.Service.ReverseLedgerEntry(actor, req.EntryID, req.Reason); err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	_ = response.JSON(w, http.StatusOK, map[string]string{"status": "reversed"})
}

// ---------- Fee Withholding (Principal Executive Toggle) ----------

type feeWithholdingRequest struct {
	Enabled bool `json:"enabled"`
}

// SetFeeWithholding handles POST /api/v1/finance/fee-withholding.
// Only the Principal can toggle this executive flag.
// When enabled, students with fee arrears cannot see their report cards on the portal.
func (h Handler) SetFeeWithholding(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req feeWithholdingRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := h.Service.SetFeeWithholding(actor, req.Enabled); err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	_ = response.JSON(w, http.StatusOK, map[string]bool{"fee_withholding_enabled": req.Enabled})
}

// ---------- Fee Waivers ----------

type grantWaiverRequest struct {
	StudentID   string `json:"student_id"`
	AmountCents int64  `json:"amount_cents"`
	Reason      string `json:"reason"`
}

// GrantFeeWaiver handles POST /api/v1/finance/waivers.
// Only the Principal can grant individual result-access exemptions (bursaries, etc.).
// A granted waiver immediately unlocks the student's report card on the portal.
func (h Handler) GrantFeeWaiver(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req grantWaiverRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	waiver := domain.FeeWaiver{
		StudentID:   strings.TrimSpace(req.StudentID),
		AmountCents: req.AmountCents,
		Reason:      strings.TrimSpace(req.Reason),
	}
	if err := h.Service.GrantFeeWaiver(actor, waiver); err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	_ = response.JSON(w, http.StatusCreated, map[string]string{"status": "waiver_granted"})
}

// ---------- Portal Result Gate ----------

// CanViewResults handles GET /api/v1/finance/results-access/{student_id}.
// The parent/student portal calls this to decide whether to show the report-card
// download button or to display the fee-arrears administrative notice instead.
func (h Handler) CanViewResults(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	studentID := r.PathValue("student_id")
	if strings.TrimSpace(studentID) == "" {
		response.Error(w, http.StatusBadRequest, "student_id is required")
		return
	}
	if !h.Service.CanAccessStudent(actor, studentID) {
		response.Error(w, http.StatusForbidden, "student results access denied")
		return
	}
	canView := h.Service.CanViewResults(studentID)
	_ = response.JSON(w, http.StatusOK, map[string]any{
		"student_id":       studentID,
		"can_view_results": canView,
	})
}

// ---------- M-Pesa C2B Callback ----------

// mpesaC2BCallback is the Safaricom Daraja C2B paybill/till notification payload.
// Safaricom sends this payload after a customer completes a pay-bill or buy-goods
// transaction. TransAmount is a string decimal (e.g. "1500.00").
type mpesaC2BCallback struct {
	TransID         string `json:"TransID"`
	TransAmount     string `json:"TransAmount"`
	BillRefNumber   string `json:"BillRefNumber"`   // student admission number
	MSISDN          string `json:"MSISDN"`          // payer phone number
	TransactionType string `json:"TransactionType"` // "Pay Bill" or "Buy Goods"
}

// MPesaC2BCallback handles POST /api/v1/finance/mpesa/callback.
// Safaricom calls this URL after every successful C2B payment. The endpoint
// uses a dedicated unguessable callback token because it cannot use an
// end-user Bearer session. A 200 response with
// ResultCode "0" acknowledges receipt; any non-zero code triggers a Daraja retry.
func (h Handler) MPesaC2BCallback(w http.ResponseWriter, r *http.Request) {
	providedToken := r.URL.Query().Get("token")
	if h.MPesaCallbackToken == "" || len(providedToken) != len(h.MPesaCallbackToken) || subtle.ConstantTimeCompare([]byte(providedToken), []byte(h.MPesaCallbackToken)) != 1 {
		response.Error(w, http.StatusUnauthorized, "invalid callback token")
		return
	}
	var cb mpesaC2BCallback
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&cb); err != nil {
		// Return 400 to signal that the payload was malformed; Daraja will not retry malformed bodies.
		response.Error(w, http.StatusBadRequest, "invalid callback payload")
		return
	}

	amountCents, err := parseDarajaAmount(cb.TransAmount)
	if err != nil {
		_ = response.JSON(w, http.StatusOK, darajaResult("C2B00016", fmt.Sprintf("invalid amount: %s", cb.TransAmount)))
		return
	}

	payment := domain.Payment{
		Reference:       strings.TrimSpace(cb.TransID),
		AdmissionNumber: strings.TrimSpace(cb.BillRefNumber),
		Phone:           strings.TrimSpace(cb.MSISDN),
		Channel:         "c2b_" + strings.ToLower(strings.ReplaceAll(cb.TransactionType, " ", "_")),
		AmountCents:     amountCents,
	}

	if err := h.Service.RecordMPesaPayment(payment); err != nil {
		// Business-logic rejection; acknowledge with a non-zero Daraja result code
		// so Safaricom marks it as processed (not retried).
		_ = response.JSON(w, http.StatusOK, darajaResult("C2B00016", err.Error()))
		return
	}

	_ = response.JSON(w, http.StatusOK, darajaResult("0", "Accepted"))
}

// ---------- STK Push ----------

type stkPushRequest struct {
	StudentID   string `json:"student_id"`
	PhoneNumber string `json:"phone_number"`
	AmountCents int64  `json:"amount_cents"`
	Description string `json:"description"`
}

// InitiateSTKPush handles POST /api/v1/finance/mpesa/stk-push.
// Parents use this from the portal to trigger a Lipa Na M-Pesa Online (STK Push)
// directly to their phone. The DarajaClient must be injected for this to work;
// when nil the handler returns 503 so the portal can degrade gracefully and
// advise the parent to use manual pay-bill instead.
func (h Handler) InitiateSTKPush(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if h.DarajaClient == nil || h.DarajaShortCode == "" {
		response.Error(w, http.StatusServiceUnavailable, "M-Pesa online payment is not yet enabled; please use the paybill number directly")
		return
	}
	var req stkPushRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	req.StudentID = strings.TrimSpace(req.StudentID)
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	if req.StudentID == "" || req.PhoneNumber == "" || req.AmountCents <= 0 {
		response.Error(w, http.StatusBadRequest, "student_id, phone_number, and positive amount_cents are required")
		return
	}
	if !h.Service.CanAccessStudent(actor, req.StudentID) {
		response.Error(w, http.StatusForbidden, "payment access denied for this student")
		return
	}

	accountRef := req.StudentID
	if desc := strings.TrimSpace(req.Description); desc != "" {
		accountRef = desc
	}

	stkResp, err := h.DarajaClient.InitiateSTKPush(r.Context(), daraja.STKPushRequest{
		BusinessShortCode: h.DarajaShortCode,
		PhoneNumber:       req.PhoneNumber,
		AccountReference:  accountRef,
		AmountCents:       req.AmountCents,
	})
	if err != nil {
		response.Error(w, http.StatusBadGateway, "M-Pesa gateway error: "+err.Error())
		return
	}
	_ = response.JSON(w, http.StatusAccepted, map[string]string{
		"checkout_request_id": stkResp.CheckoutRequestID,
		"merchant_request_id": stkResp.RequestID,
		"status":              "stk_push_initiated",
	})
}

// ---------- Unmatched Payments ----------

// ListUnmatchedPayments handles GET /api/v1/finance/mpesa/unmatched.
// Returns all M-Pesa payments that could not be auto-matched to a student
// (e.g. unknown or misspelled admission number) and are awaiting manual resolution.
func (h Handler) ListUnmatchedPayments(w http.ResponseWriter, r *http.Request) {
	_, ok := actorFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	payments := h.Service.UnmatchedPayments()
	_ = response.JSON(w, http.StatusOK, map[string]any{
		"unmatched": payments,
		"count":     len(payments),
	})
}

type resolveUnmatchedRequest struct {
	Reference string `json:"reference"`
	StudentID string `json:"student_id"`
}

// ResolveUnmatchedPayment handles POST /api/v1/finance/mpesa/unmatched/resolve.
// Finance staff manually link an unmatched payment to the correct student and
// trigger the SMS payment receipt.
func (h Handler) ResolveUnmatchedPayment(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req resolveUnmatchedRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	req.Reference = strings.TrimSpace(req.Reference)
	req.StudentID = strings.TrimSpace(req.StudentID)
	if err := h.Service.ResolveUnmatchedPayment(actor, req.Reference, req.StudentID); err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	_ = response.JSON(w, http.StatusOK, map[string]string{"status": "payment_allocated"})
}

// ---------- helpers ----------

// parseDarajaAmount converts a Daraja decimal string (e.g. "1500.00") to integer cents.
func parseDarajaAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("amount is empty")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parse amount %q: %w", s, err)
	}
	if f <= 0 {
		return 0, fmt.Errorf("amount must be positive, got %s", s)
	}
	return int64(f * 100), nil
}

// darajaResult builds the standard Safaricom C2B acknowledgement envelope.
func darajaResult(code, desc string) map[string]string {
	return map[string]string{"ResultCode": code, "ResultDesc": desc}
}
