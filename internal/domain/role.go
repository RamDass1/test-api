package domain

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	}
	return false
}

func (r Role) Assignable() bool {
	return r == RoleAdmin || r == RoleMember
}
func (r Role) CanManageMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}

func (r Role) CanEditAnyTask() bool {
	return r == RoleOwner || r == RoleAdmin
}

func (r Role) CanViewStats() bool {
	return r == RoleOwner || r == RoleAdmin
}
