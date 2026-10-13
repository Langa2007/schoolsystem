package sms

import "elimu-core/backend/internal/domain"

type MemoryAdapter = domain.MemoryNotifier

func NewMemoryAdapter() *MemoryAdapter {
	return &MemoryAdapter{}
}
