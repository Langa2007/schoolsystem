package daraja

import "context"

type STKPushRequest struct {
	BusinessShortCode, PhoneNumber, AccountReference string
	AmountCents                                      int64
}

type STKPushResponse struct {
	RequestID, CheckoutRequestID string
}

type Client interface {
	InitiateSTKPush(context.Context, STKPushRequest) (STKPushResponse, error)
}
