package auth

import (
	"context"
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
	user, err := VerifyToken(session.AccessToken, service.Secret, time.Now())
	if err != nil || user.ID != principal.ID || user.SchoolID != principal.SchoolID {
		t.Fatalf("unexpected verified user: %#v, %v", user, err)
	}
	if _, err := VerifyToken(session.AccessToken, []byte("01234567890123456789012345678902"), time.Now()); err == nil {
		t.Fatal("expected a token signed with another secret to fail")
	}
}
