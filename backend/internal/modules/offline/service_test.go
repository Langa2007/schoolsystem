package offline

import (
	"testing"
)

func TestSyncIsIdempotentAndDetectsConflicts(t *testing.T) {
	applied := 0
	service := NewService(func(Operation) error { applied++; return nil })
	operation := Operation{ID: "op-1", TenantID: "school-1", ActorID: "teacher-1", Entity: "attendance", Action: "upsert", Version: 1, Payload: []byte(`{"student_id":"s1"}`)}
	first, err := service.Apply(operation)
	if err != nil || first.Status != "applied" {
		t.Fatalf("first apply: %#v %v", first, err)
	}
	retry, err := service.Apply(operation)
	if err != nil || retry.Status != "applied" || applied != 1 {
		t.Fatalf("retry should be idempotent: %#v %v applied=%d", retry, err, applied)
	}
	conflict, err := service.Apply(Operation{ID: "op-2", TenantID: "school-1", ActorID: "teacher-1", Entity: "attendance", Action: "upsert", Version: 3, Payload: []byte(`{}`)})
	if err != nil || !conflict.Conflict {
		t.Fatalf("expected version conflict: %#v %v", conflict, err)
	}
}
