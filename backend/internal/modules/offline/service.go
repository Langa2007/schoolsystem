package offline

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

type ApplyFunc func(Operation) error

type Service struct {
	mu         sync.Mutex
	operations map[string]Result
	versions   map[string]int64
	apply      ApplyFunc
}

func NewService(apply ApplyFunc) *Service {
	return &Service{operations: map[string]Result{}, versions: map[string]int64{}, apply: apply}
}

func (s *Service) Apply(operation Operation) (Result, error) {
	if operation.ID == "" || operation.TenantID == "" || operation.ActorID == "" || operation.Entity == "" || operation.Action == "" {
		return Result{}, errors.New("operation id, tenant, actor, entity, and action are required")
	}
	if !json.Valid(operation.Payload) {
		return Result{}, errors.New("operation payload must be valid JSON")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if previous, ok := s.operations[operation.ID]; ok {
		return previous, nil
	}
	key := operation.TenantID + ":" + operation.Entity
	current := s.versions[key]
	if operation.Version != current+1 && current != 0 {
		result := Result{OperationID: operation.ID, Status: "conflict", Conflict: true, Message: fmt.Sprintf("expected version %d, got %d", current+1, operation.Version)}
		s.operations[operation.ID] = result
		return result, nil
	}
	if s.apply == nil {
		return Result{}, errors.New("offline operation applier is not configured")
	}
	if err := s.apply(operation); err != nil {
		return Result{}, fmt.Errorf("apply offline operation: %w", err)
	}
	s.versions[key] = operation.Version
	result := Result{OperationID: operation.ID, Status: "applied", Message: "operation applied"}
	s.operations[operation.ID] = result
	return result, nil
}
