package communication

import (
	"errors"
	"fmt"
	"sync"
)

type Service struct {
	mu       sync.Mutex
	notifier Notifier
}

func NewService(notifier Notifier) *Service { return &Service{notifier: notifier} }

func (s *Service) SendBulk(message Message) (int, error) {
	if message.ID == "" || message.Body == "" || len(message.Recipients) == 0 {
		return 0, errors.New("message id, body, and recipients are required")
	}
	if s.notifier == nil {
		return 0, errors.New("SMS notifier is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sent := 0
	for _, recipient := range message.Recipients {
		if recipient.Phone == "" {
			continue
		}
		if err := s.notifier.SendSMS(recipient.Phone, message.Body); err != nil {
			return sent, fmt.Errorf("send to %s: %w", recipient.Phone, err)
		}
		sent++
	}
	return sent, nil
}
