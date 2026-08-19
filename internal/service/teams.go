package service

import (
	"context"
	"errors"
	"strings"

	"github.com/RamDass1/test-api/internal/domain"
)

type InviteRequest struct {
	Email  string
	UserID int64
	Role   domain.Role
}

func (s *Service) CreateTeam(ctx context.Context, userID int64, name string) (domain.Team, error) {
	teamName, err := domain.NormalizeTeamName(name)
	if err != nil {
		return domain.Team{}, err
	}
	team, err := s.db.CreateTeam(ctx, teamName, userID)
	if errors.Is(err, domain.ErrUnknownID) {
		return domain.Team{}, domain.Unauthorized("authenticated user no longer exists")
	}
	if err != nil {
		return domain.Team{}, domain.Internal(err, "create team")
	}
	return team, nil
}

func (s *Service) ListTeams(ctx context.Context, userID int64) ([]domain.Team, error) {
	teams, err := s.db.TeamsForUser(ctx, userID)
	if err != nil {
		return nil, domain.Internal(err, "list teams")
	}
	return teams, nil
}

func (s *Service) InviteMember(ctx context.Context, actorID, teamID int64, req InviteRequest) (domain.TeamMember, error) {
	if req.Role == "" {
		req.Role = domain.RoleMember
	}
	if !req.Role.Valid() {
		return domain.TeamMember{}, domain.Invalid("unknown role %q", req.Role)
	}

	hasID := req.UserID > 0
	hasEmail := strings.TrimSpace(req.Email) != ""
	if !hasID && !hasEmail {
		return domain.TeamMember{}, domain.Invalid("user_id or email is required")
	}
	if hasID && hasEmail {
		return domain.TeamMember{}, domain.Invalid("provide user_id or email, not both")
	}

	actor, err := membership(ctx, s.db, teamID, actorID)
	if err != nil {
		return domain.TeamMember{}, err
	}
	if err := domain.AuthorizeInvite(actor, req.Role); err != nil {
		return domain.TeamMember{}, err
	}

	invitee, err := s.resolveInvitee(ctx, req)
	if err != nil {
		return domain.TeamMember{}, err
	}

	err = s.db.AddMember(ctx, teamID, invitee.ID, req.Role)
	if errors.Is(err, domain.ErrAlreadyExists) {
		return domain.TeamMember{}, domain.Conflict("user %d is already member of team %d", invitee.ID, teamID)
	}
	if err != nil {
		return domain.TeamMember{}, domain.Internal(err, "add team member")
	}

	return domain.TeamMember{
		TeamID: teamID,
		UserID: invitee.ID,
		Role:   req.Role,
		Email:  invitee.Email,
		Name:   invitee.Name,
	}, nil
}

func (s *Service) resolveInvitee(ctx context.Context, req InviteRequest) (domain.User, error) {
	if req.UserID > 0 {
		user, err := s.db.UserByID(ctx, req.UserID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, domain.NotFound("user %d not found", req.UserID)
		}
		if err != nil {
			return domain.User{}, domain.Internal(err, "load invite")
		}
		return user, nil
	}

	email, err := domain.NormalizeEmail(req.Email)
	if err != nil {
		return domain.User{}, err
	}
	user, err := s.db.UserByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, domain.NotFound("no user registered with this email")
	}
	if err != nil {
		return domain.User{}, domain.Internal(err, "load invite")
	}
	return user, nil
}

func (s *Service) ChangeMemberRole(ctx context.Context, actorID, teamID, targetID int64, role domain.Role) (domain.TeamMember, error) {
	if !role.Valid() {
		return domain.TeamMember{}, domain.Invalid("unknown role %q", role)
	}

	actor, err := membership(ctx, s.db, teamID, actorID)
	if err != nil {
		return domain.TeamMember{}, err
	}

	target, err := s.db.Membership(ctx, teamID, targetID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.TeamMember{}, domain.NotFound("user %d is not a member of team %d", targetID, teamID)
	}
	if err != nil {
		return domain.TeamMember{}, domain.Internal(err, "load target membership")
	}

	if err := domain.AuthorizeRoleChange(actor, target, role); err != nil {
		return domain.TeamMember{}, err
	}
	if err := s.db.UpdateMemberRole(ctx, teamID, targetID, role); err != nil {
		return domain.TeamMember{}, domain.Internal(err, "update member role")
	}

	target.Role = role
	return target, nil
}

func membership(ctx context.Context, db DB, teamID, userID int64) (domain.Actor, error) {
	member, err := db.Membership(ctx, teamID, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Actor{}, domain.NotFound("team %d not found", teamID)
	}
	if err != nil {
		return domain.Actor{}, domain.Internal(err, "resolve team membership")
	}
	return domain.Actor{UserID: userID, Role: member.Role}, nil
}
