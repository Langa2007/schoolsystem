package offline

import "time"

type Operation struct {
	ID, TenantID, ActorID, Entity, Action string
	Version                               int64
	Payload                               []byte
	CreatedAt                             time.Time
}

type Result struct {
	OperationID string
	Status      string
	Conflict    bool
	Message     string
}

type Store interface {
	Apply(Operation) (Result, error)
}
