package phaseone

import (
	"testing"

	"elimu-core/backend/internal/domain"
	"elimu-core/backend/internal/pkg/sms"
)

func TestFoundationServiceCanInitialize(t *testing.T) {
	_, err := domain.NewService(
		domain.SchoolSettings{ID: "school-1", Name: "Elimu High", Mode: domain.MixedMode},
		sms.NewMemoryAdapter(),
	)
	if err != nil {
		t.Fatal(err)
	}
}
