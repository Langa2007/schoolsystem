package tenancy

import (
	"errors"
	"sync"
)

type Tenant struct {
	ID, Name, ClusterID string
	Active              bool
}

type Service struct {
	mu      sync.RWMutex
	tenants map[string]Tenant
}

func NewService() *Service { return &Service{tenants: map[string]Tenant{}} }

func (s *Service) Register(tenant Tenant) error {
	if tenant.ID == "" || tenant.Name == "" {
		return errors.New("tenant id and name are required")
	}
	tenant.Active = true
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tenants[tenant.ID]; exists {
		return errors.New("tenant already exists")
	}
	s.tenants[tenant.ID] = tenant
	return nil
}

func (s *Service) Resolve(id string) (Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tenant, ok := s.tenants[id]
	if !ok || !tenant.Active {
		return Tenant{}, errors.New("tenant not found or inactive")
	}
	return tenant, nil
}

func (s *Service) SameTenant(requestTenant, resourceTenant string) error {
	if requestTenant == "" || resourceTenant == "" || requestTenant != resourceTenant {
		return errors.New("tenant isolation violation")
	}
	return nil
}
