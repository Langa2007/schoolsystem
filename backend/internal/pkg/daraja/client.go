// Package daraja provides the Safaricom M-Pesa Daraja API client boundary.
// The Client interface is the production boundary; HTTPClient is a concrete
// implementation that carries out OAuth token acquisition and STK Push requests
// against the Safaricom sandbox or production endpoints.
package daraja

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// STKPushRequest carries the parameters for a Lipa Na M-Pesa Online (STK Push) request.
type STKPushRequest struct {
	// BusinessShortCode is the paybill or till number configured for the school.
	BusinessShortCode string
	// PhoneNumber is the subscriber's Safaricom number in international format (e.g. 254700123456).
	PhoneNumber string
	// AccountReference is displayed on the customer's phone; typically the student admission number.
	AccountReference string
	// AmountCents is the charge amount in minor currency units (Kenyan cents).
	AmountCents int64
}

// STKPushResponse is returned by Safaricom after a successful STK push initiation.
type STKPushResponse struct {
	// RequestID is the MerchantRequestID from the Daraja response.
	RequestID string
	// CheckoutRequestID is the unique identifier for the push request, used to
	// query the status via the Query API.
	CheckoutRequestID string
}

// Client is the production boundary for M-Pesa operations.
// A nil implementation means the M-Pesa gateway is not yet configured;
// callers should degrade gracefully (e.g. return HTTP 503).
type Client interface {
	InitiateSTKPush(context.Context, STKPushRequest) (STKPushResponse, error)
}

// Endpoints for Daraja sandbox and production environments.
const (
	SandboxBaseURL    = "https://sandbox.safaricom.co.ke"
	ProductionBaseURL = "https://api.safaricom.co.ke"
)

// HTTPClient is a concrete Daraja Client backed by the standard net/http package.
// It caches the OAuth bearer token so that repeated calls within the same TTL
// do not re-authenticate on every request.
type HTTPClient struct {
	BaseURL          string
	ConsumerKey      string
	ConsumerSecret   string
	PassKey          string // Lipa Na M-Pesa Online passkey from Safaricom portal
	CallbackURL      string // Publicly accessible HTTPS URL for STK Push results

	httpClient *http.Client

	mu           sync.Mutex
	cachedToken  string
	tokenExpires time.Time
}

// NewHTTPClient constructs a production-ready Daraja HTTPClient.
// Use SandboxBaseURL during development and ProductionBaseURL for live deployments.
func NewHTTPClient(baseURL, consumerKey, consumerSecret, passKey, callbackURL string) *HTTPClient {
	return &HTTPClient{
		BaseURL:        baseURL,
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
		PassKey:        passKey,
		CallbackURL:    callbackURL,
		httpClient:     &http.Client{Timeout: 15 * time.Second},
	}
}

// token returns a valid OAuth2 bearer token, fetching a new one if the cached
// token has expired or is absent.
func (c *HTTPClient) token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cachedToken != "" && time.Now().Before(c.tokenExpires) {
		return c.cachedToken, nil
	}
	creds := base64.StdEncoding.EncodeToString([]byte(c.ConsumerKey + ":" + c.ConsumerSecret))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/oauth/v1/generate?grant_type=client_credentials", nil)
	if err != nil {
		return "", fmt.Errorf("daraja: build token request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+creds)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("daraja: fetch token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("daraja: token endpoint returned %d", resp.StatusCode)
	}
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("daraja: decode token response: %w", err)
	}
	c.cachedToken = body.AccessToken
	// Safaricom tokens are valid for 3600 seconds; subtract 60s for safety margin.
	c.tokenExpires = time.Now().Add(3540 * time.Second)
	return c.cachedToken, nil
}

// InitiateSTKPush sends a Lipa Na M-Pesa Online STK Push request to Safaricom.
// The customer receives a USSD prompt on their phone to enter their M-Pesa PIN.
// After the customer approves or cancels, Safaricom posts the result to CallbackURL.
func (c *HTTPClient) InitiateSTKPush(ctx context.Context, r STKPushRequest) (STKPushResponse, error) {
	token, err := c.token(ctx)
	if err != nil {
		return STKPushResponse{}, err
	}

	// Timestamp in yyyyMMddHHmmss format as required by the Daraja API.
	timestamp := time.Now().Format("20060102150405")
	// The password is Base64(BusinessShortCode + PassKey + Timestamp).
	password := base64.StdEncoding.EncodeToString(
		[]byte(r.BusinessShortCode + c.PassKey + timestamp),
	)

	// Amount is in whole Kenya Shillings; convert from cents by integer division.
	amountKES := r.AmountCents / 100
	if amountKES < 1 {
		amountKES = 1
	}

	payload := map[string]any{
		"BusinessShortCode": r.BusinessShortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerPayBillOnline",
		"Amount":            fmt.Sprintf("%d", amountKES),
		"PartyA":            r.PhoneNumber,
		"PartyB":            r.BusinessShortCode,
		"PhoneNumber":       r.PhoneNumber,
		"CallBackURL":       c.CallbackURL,
		"AccountReference":  r.AccountReference,
		"TransactionDesc":   "School fees payment",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return STKPushResponse{}, fmt.Errorf("daraja: marshal stk push request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/mpesa/stkpush/v1/processrequest",
		bytes.NewReader(body))
	if err != nil {
		return STKPushResponse{}, fmt.Errorf("daraja: build stk push request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return STKPushResponse{}, fmt.Errorf("daraja: stk push request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		MerchantRequestID   string `json:"MerchantRequestID"`
		CheckoutRequestID   string `json:"CheckoutRequestID"`
		ResponseCode        string `json:"ResponseCode"`
		ResponseDescription string `json:"ResponseDescription"`
		CustomerMessage     string `json:"CustomerMessage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return STKPushResponse{}, fmt.Errorf("daraja: decode stk push response: %w", err)
	}
	if result.ResponseCode != "0" {
		return STKPushResponse{}, fmt.Errorf("daraja: stk push rejected: %s", result.ResponseDescription)
	}
	return STKPushResponse{
		RequestID:         result.MerchantRequestID,
		CheckoutRequestID: result.CheckoutRequestID,
	}, nil
}
