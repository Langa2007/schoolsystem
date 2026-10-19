package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"elimu-core/backend/internal/domain"
	"elimu-core/backend/internal/modules/auth"
	"elimu-core/backend/internal/modules/school_config"
	"elimu-core/backend/internal/pkg/sms"
)

func TestRoutesProtectFinanceAndRegisterPhaseStatusAtStartup(t *testing.T) {
	service, err := school_config.NewService(school_config.SchoolSettings{ID: "school-1", Name: "Elimu", Mode: domain.DayOnly}, sms.NewMemoryAdapter())
	if err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Domain: service, Secret: []byte("01234567890123456789012345678901"), TTL: time.Hour}
	handler := routes(service, authService, nil, nil, "", "", "http://localhost:3000")

	private := httptest.NewRecorder()
	handler.ServeHTTP(private, httptest.NewRequest(http.MethodPost, "/api/v1/finance/vote-heads", nil))
	if private.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated finance request status=%d", private.Code)
	}
	callback := httptest.NewRecorder()
	handler.ServeHTTP(callback, httptest.NewRequest(http.MethodPost, "/api/v1/finance/mpesa/callback", strings.NewReader(`{}`)))
	if callback.Code != http.StatusUnauthorized {
		t.Fatalf("unconfigured payment callback status=%d", callback.Code)
	}

	phaseFive := httptest.NewRecorder()
	handler.ServeHTTP(phaseFive, httptest.NewRequest(http.MethodGet, "/api/v1/phase-five", nil))
	if phaseFive.Code != http.StatusOK {
		t.Fatalf("phase-five status endpoint status=%d", phaseFive.Code)
	}
}

func TestCorsAllowsConfiguredFrontendAndRejectsUnknownOrigin(t *testing.T) {
	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), "http://localhost:3000")
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/students", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", "GET")
	allowed := httptest.NewRecorder()
	handler.ServeHTTP(allowed, request)
	if allowed.Code != http.StatusNoContent || allowed.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("unexpected CORS response: %d %#v", allowed.Code, allowed.Header())
	}
	request = httptest.NewRequest(http.MethodOptions, "/api/v1/students", nil)
	request.Header.Set("Origin", "https://attacker.example")
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, request)
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("unconfigured origin status=%d", blocked.Code)
	}
}

func TestAuthenticatedStudentAPI(t *testing.T) {
	service, err := school_config.NewService(school_config.SchoolSettings{ID: "school-1", Name: "Elimu", Mode: domain.DayOnly}, sms.NewMemoryAdapter())
	if err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: "principal", SchoolID: "school-1", Name: "Principal", Roles: []domain.Role{domain.RolePrincipal}}
	if err := service.AddUser(admin); err != nil {
		t.Fatal(err)
	}
	if err := service.SetPassword(admin, admin.ID, "SecurePass1"); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Domain: service, Secret: []byte("01234567890123456789012345678901"), TTL: time.Hour}
	session, err := authService.Authenticate(context.Background(), admin.ID, "SecurePass1")
	if err != nil {
		t.Fatal(err)
	}
	handler := routes(service, authService, nil, nil, "", "", "http://localhost:3000")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/students", strings.NewReader(`{"id":"student-1","admission_number":"A-1","name":"Amina","class_stream":"Grade 10 A"}`))
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)
	request.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, request)
	if created.Code != http.StatusCreated {
		t.Fatalf("create student status=%d body=%s", created.Code, created.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v1/students", nil)
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)
	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, request)
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), "A-1") {
		t.Fatalf("student listing response=%d %s", listed.Code, listed.Body.String())
	}
}
