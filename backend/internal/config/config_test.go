package config

import (
	"testing"
	"time"
)

func TestProductionConfigRequiresSecretsAndDatabase(t *testing.T) {
	config := Config{
		HTTPAddress: ":8080", Environment: "production", SchoolID: "school-1",
		SchoolName: "Elimu", SchoolMode: "day", SessionTTL: time.Hour, ShutdownTimeout: time.Second,
	}
	if err := config.Validate(); err == nil {
		t.Fatal("expected production configuration validation to fail")
	}
}

func TestDevelopmentConfigValidation(t *testing.T) {
	config := Config{
		HTTPAddress: ":8080", Environment: "development", SchoolID: "school-1",
		SchoolName: "Elimu", SchoolMode: "day", SessionTTL: time.Hour, ShutdownTimeout: time.Second,
	}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
}
