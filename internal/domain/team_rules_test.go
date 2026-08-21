package domain_test

import (
	"errors"
	"testing"

	"github.com/RamDass1/test-api/internal/domain"
)

const (
	owner    = 1
	admin    = 2
	creator  = 3
	assignee = 4
	stranger = 5
)

func TestAuthorizeInvite(t *testing.T) {
	tests := []struct {
		name    string
		role    domain.Role
		invited domain.Role
		wantErr domain.Code
	}{
		{name: "owner invites admin", role: domain.RoleOwner, invited: domain.RoleAdmin},
		{name: "admin invites member", role: domain.RoleAdmin, invited: domain.RoleMember},
		{name: "member cannot invite", role: domain.RoleMember, invited: domain.RoleMember, wantErr: domain.CodeForbidden},
		{name: "ownership cannot be handed out", role: domain.RoleOwner, invited: domain.RoleOwner, wantErr: domain.CodeValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.AuthorizeInvite(domain.Actor{UserID: owner, Role: tt.role}, tt.invited)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected invite to be allowed, got %v", err)
				}
				return
			}
			assertCode(t, err, tt.wantErr)
		})
	}
}

func TestAuthorizeRoleChange(t *testing.T) {
	target := domain.TeamMember{TeamID: 1, UserID: assignee, Role: domain.RoleMember}

	t.Run("owner promotes member", func(t *testing.T) {
		if err := domain.AuthorizeRoleChange(domain.Actor{UserID: owner, Role: domain.RoleOwner}, target, domain.RoleAdmin); err != nil {
			t.Fatalf("expected promotion to be allowed, got %v", err)
		}
	})

	t.Run("admin cannot change roles", func(t *testing.T) {
		err := domain.AuthorizeRoleChange(domain.Actor{UserID: admin, Role: domain.RoleAdmin}, target, domain.RoleMember)
		assertCode(t, err, domain.CodeForbidden)
	})

	t.Run("the owner cannot be demoted", func(t *testing.T) {
		ownerMember := domain.TeamMember{TeamID: 1, UserID: owner, Role: domain.RoleOwner}
		err := domain.AuthorizeRoleChange(domain.Actor{UserID: owner, Role: domain.RoleOwner}, ownerMember, domain.RoleMember)
		assertCode(t, err, domain.CodeForbidden)
	})

	t.Run("ownership cannot be granted", func(t *testing.T) {
		err := domain.AuthorizeRoleChange(domain.Actor{UserID: owner, Role: domain.RoleOwner}, target, domain.RoleOwner)
		assertCode(t, err, domain.CodeValidation)
	})
}

func TestAuthorizeStatsAccess(t *testing.T) {
	if err := domain.AuthorizeStatsAccess(domain.Actor{UserID: admin, Role: domain.RoleAdmin}); err != nil {
		t.Fatalf("admin should be allowed to read report, got %v", err)
	}
	assertCode(t, domain.AuthorizeStatsAccess(domain.Actor{UserID: stranger, Role: domain.RoleMember}), domain.CodeForbidden)
}

func assertCode(t *testing.T, err error, want domain.Code) {
	t.Helper()

	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected domain error with code %q, got %v", want, err)
	}
	if domainErr.Code != want {
		t.Fatalf("expected code %q, got %q (%s)", want, domainErr.Code, domainErr.Message)
	}
}
