package main

import (
	"encoding/json"
	"log"
	"net/http"

	"elimu-core/backend/internal/config"
	"elimu-core/backend/internal/modules/school_config"
	"elimu-core/backend/internal/pkg/sms"
)

func main() {
	cfg := config.FromEnvironment()
	service, err := school_config.NewService(school_config.SchoolSettings{
		ID: cfg.SchoolID, Name: cfg.SchoolName,
		Mode: school_config.SchoolMode(cfg.SchoolMode), PublicSchool: cfg.PublicSchool,
	}, sms.NewMemoryAdapter())
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
			"note":        "Phase 1 domain behavior is implemented behind the structured module facades; persistence and production gateways are deployment work.",
		})
	})
	mux.HandleFunc("GET /api/v1/phase-two", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"curriculum":  "pure_cbc",
			"implemented": []string{"grade_10_to_12_pathways", "competency_rubrics", "exam_mark_validation", "submission_locking", "admission_sorted_verification", "dos_principal_approval_gates", "correction_audit_trail"},
			"open":        []string{"concurrent_pdf_report_worker"},
		})
	})
	log.Printf("ElimuCore Phase 1 API listening on %s", cfg.HTTPAddress)
	log.Fatal(http.ListenAndServe(cfg.HTTPAddress, mux))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}
