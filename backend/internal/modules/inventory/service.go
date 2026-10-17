package inventory

import (
	"errors"
	"sync"
)

type Service struct {
	mu           sync.Mutex
	items        map[string]Item
	transactions []Transaction
}

func NewService() *Service {
	return &Service{items: map[string]Item{}}
}

func (s *Service) AddItem(item Item) error {
	if item.ID == "" || item.SchoolID == "" || item.Name == "" || item.Quantity < 0 || item.ReorderLevel < 0 {
		return errors.New("invalid inventory item")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[item.ID]; exists {
		return errors.New("inventory item already exists")
	}
	s.items[item.ID] = item
	return nil
}

func (s *Service) Adjust(transaction Transaction) error {
	if transaction.ID == "" || transaction.ItemID == "" || transaction.ActorID == "" || transaction.Quantity <= 0 {
		return errors.New("valid inventory transaction is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[transaction.ItemID]
	if !ok {
		return errors.New("inventory item not found")
	}
	switch transaction.Type {
	case "receive":
		item.Quantity += transaction.Quantity
	case "issue":
		if item.Quantity < transaction.Quantity {
			return errors.New("insufficient inventory")
		}
		item.Quantity -= transaction.Quantity
	default:
		return errors.New("transaction type must be receive or issue")
	}
	s.items[item.ID] = item
	s.transactions = append(s.transactions, transaction)
	return nil
}
