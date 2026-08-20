package domain

func AuthorizeInvite(actor Actor, role Role) error {
	if !actor.Role.CanManageMembers() {
		return Forbidden("only the team owner or an admin may invite members")
	}
	if !role.Assignable() {
		return Invalid("role %q cannot be assigned through an invite", role)
	}
	return nil
}

func AuthorizeRoleChange(actor Actor, target TeamMember, newRole Role) error {
	if actor.Role != RoleOwner {
		return Forbidden("only the team owner may change member roles")
	}
	if target.Role == RoleOwner {
		return Forbidden("the team owner's role cannot be changed")
	}
	if !newRole.Assignable() {
		return Invalid("role %q cannot be assigned", newRole)
	}
	return nil
}

func AuthorizeStatsAccess(actor Actor) error {
	if !actor.Role.CanViewStats() {
		return Forbidden("only the team owner or an admin may view team analytics")
	}
	return nil
}
