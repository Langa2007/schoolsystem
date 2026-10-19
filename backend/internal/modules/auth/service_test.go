package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"elimu-core/backend/internal/domain"
	"elimu-core/backend/internal/pkg/sms"
)

func TestSessionTokenIsSignedAndTenantBound(t *testing.T) {
	domainService, err := domain.NewService(domain.SchoolSettings{ID: "school-1", Name: "Elimu", Mode: domain.DayOnly}, sms.NewMemoryAdapter())
	if err != nil {
		t.Fatal(err)
	}
	principal := domain.User{ID: "principal", SchoolID: "school-1", Name: "Principal", Roles: []domain.Role{domain.RolePrincipal}}
	if err := domainService.AddUser(principal); err != nil {
		t.Fatal(err)
	}
	if err := domainService.SetPassword(principal, principal.ID, "SecurePass1"); err != nil {
		t.Fatal(err)
	}
	service := Service{Domain: domainService, Secret: []byte("01234567890123456789012345678901"), TTL: time.Hour}
	session, err := service.Authenticate(context.Background(), "principal", "SecurePass1")
	if err != nil {
		t.Fatal(err)
	}
	if !session.MustChangePassword {
		t.Fatal("provisioned account should require a first-login password change")
	}
	if err := domainService.ChangePassword(principal, "SecurePass1", "NewSecurePass2"); err != nil {
		t.Fatal(err)
	}
	session, err = service.Authenticate(context.Background(), "principal", "NewSecurePass2")
	if err != nil || session.MustChangePassword {
		t.Fatalf("password change state not reflected in login: %#v %v", session, err)
	}
	user, err := VerifyToken(session.AccessToken, service.Secret, time.Now())
	if err != nil || user.ID != principal.ID || user.SchoolID != principal.SchoolID {
		t.Fatalf("unexpected verified user: %#v, %v", user, err)
	}
	if _, err := VerifyToken(session.AccessToken, []byte("01234567890123456789012345678902"), time.Now()); err == nil {
		t.Fatal("expected a token signed with another secret to fail")
	}
}

func TestMiddlewareRejectsMissingTokenAndAttachesCurrentUser(t *testing.T) {
	domainService, err := domain.NewService(domain.SchoolSettings{ID: "school-1", Name: "Elimu", Mode: domain.DayOnly}, sms.NewMemoryAdapter())
	if err != nil {
		t.Fatal(err)
	}
	principal := domain.User{ID: "principal", SchoolID: "school-1", Name: "Principal", Roles: []domain.Role{domain.RolePrincipal}}
	if err := domainService.AddUser(principal); err != nil {
		t.Fatal(err)
	}
	service := &Service{Domain: domainService, Secret: []byte("01234567890123456789012345678901"), TTL: time.Hour}
	session, err := service.Authenticate(context.Background(), "principal", "SecurePass1")
	if err == nil {
		t.Fatal("expected account without password to reject login")
	}
	if err := domainService.SetPassword(principal, principal.ID, "SecurePass1"); err != nil {
		t.Fatal(err)
	}
	session, err = service.Authenticate(context.Background(), "principal", "SecurePass1")
	if err != nil {
		t.Fatal(err)
	}
	handler := service.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := ActorFromContext(r.Context())
		if !ok || actor.ID != principal.ID || actor.Roles[0] != domain.RolePrincipal {
			t.Fatalf("missing current actor: %#v", actor)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d", missing.Code)
	}
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)
	valid := httptest.NewRecorder()
	handler.ServeHTTP(valid, request)
	if valid.Code != http.StatusNoContent {
		t.Fatalf("valid token status = %d", valid.Code)
	}
}
