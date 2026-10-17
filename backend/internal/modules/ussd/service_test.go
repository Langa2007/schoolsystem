package ussd

import "testing"

type gateway struct{}

func (gateway) Balance(string) (int64, error)     { return 125000, nil }
func (gateway) Attendance(string) (string, error) { return "No new alerts", nil }

func TestUSSDFeeBalance(t *testing.T) {
	message, err := (Service{Gateway: gateway{}}).Handle("session-1", "+254700000001", "student-1*1")
	if err != nil || message != "END Fee balance: KES 1250" {
		t.Fatalf("unexpected USSD response: %q, %v", message, err)
	}
}
