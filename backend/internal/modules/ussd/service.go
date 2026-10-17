package ussd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Gateway interface {
	Balance(studentID string) (int64, error)
	Attendance(studentID string) (string, error)
}

type Service struct{ Gateway Gateway }

func (s Service) Handle(sessionID, phone, text string) (string, error) {
	if sessionID == "" || phone == "" {
		return "", errors.New("session and phone are required")
	}
	parts := strings.Split(strings.TrimSpace(text), "*")
	if len(parts) == 1 && parts[0] == "" {
		return "CON Welcome to ElimuCore\n1. Fee balance\n2. Attendance alerts\n0. Exit", nil
	}
	if len(parts) < 2 || s.Gateway == nil {
		return "END Invalid selection", nil
	}
	studentID := parts[0]
	switch parts[1] {
	case "1":
		balance, err := s.Gateway.Balance(studentID)
		if err != nil {
			return "", fmt.Errorf("fee balance: %w", err)
		}
		return "END Fee balance: KES " + strconv.FormatInt(balance/100, 10), nil
	case "2":
		alerts, err := s.Gateway.Attendance(studentID)
		if err != nil {
			return "", fmt.Errorf("attendance alerts: %w", err)
		}
		return "END Attendance: " + alerts, nil
	default:
		return "END Invalid selection", nil
	}
}
