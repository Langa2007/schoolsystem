package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"elimu-core/backend/internal/phaseone"
)

func main() {
	notifier := &phaseone.MemoryNotifier{}
	service, err := phaseone.NewService(phaseone.SchoolSettings{
		ID: "default-school", Name: env("SCHOOL_NAME", "ElimuCore Demo School"),
		Mode:         phaseone.SchoolMode(env("SCHOOL_MODE", string(phaseone.DayOnly))),
		PublicSchool: env("PUBLIC_SCHOOL", "true") == "true",
	}, notifier)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/settings", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, service.Settings())
	})
	mux.HandleFunc("GET /api/v1/phase-one", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"implemented": []string{"institutional_settings", "rbac_role_union", "student_staff_registers", "morning_roll", "lesson_attendance", "four_pm_reconciliation", "sms_alert_port"},
			"note":        "Use the phaseone service package to add authenticated persistence and production gateways.",
		})
	})
	log.Printf("ElimuCore Phase 1 API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}
