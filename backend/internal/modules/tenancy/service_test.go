package tenancy

import "testing"

func TestTenantIsolation(t *testing.T) {
	service := NewService()
	if err := service.Register(Tenant{ID: "school-a", Name: "A"}); err != nil {
		t.Fatal(err)
	}
	if err := service.SameTenant("school-a", "school-b"); err == nil {
		t.Fatal("expected cross-tenant access to be rejected")
	}
}
