package httpapi

import (
	"net/http"
	"strconv"

	"github.com/RamDass1/test-api/internal/domain"
	"github.com/RamDass1/test-api/internal/service"
)

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.Invalid("%s must be positive integer", name)
	}
	return id, nil
}

type memberResponse struct {
	TeamID int64       `json:"team_id"`
	UserID int64       `json:"user_id"`
	Role   domain.Role `json:"role"`
	Email  string      `json:"email,omitempty"`
	Name   string      `json:"name,omitempty"`
}

func toMemberResponse(m domain.TeamMember) memberResponse {
	return memberResponse{TeamID: m.TeamID, UserID: m.UserID, Role: m.Role, Email: m.Email, Name: m.Name}
}

func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	team, err := s.svc.CreateTeam(r.Context(), userID(r.Context()), req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, team)
}

func (s *Server) handleListTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := s.svc.ListTeams(r.Context(), userID(r.Context()))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": teams})
}

func (s *Server) handleInvite(w http.ResponseWriter, r *http.Request) {
	teamID, err := pathID(r, "team_id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Email  string      `json:"email"`
		UserID int64       `json:"user_id"`
		Role   domain.Role `json:"role"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	member, err := s.svc.InviteMember(r.Context(), userID(r.Context()), teamID, service.InviteRequest{
		Email:  req.Email,
		UserID: req.UserID,
		Role:   req.Role,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMemberResponse(member))
}

func (s *Server) handleChangeRole(w http.ResponseWriter, r *http.Request) {
	teamID, err := pathID(r, "team_id")
	if err != nil {
		writeError(w, err)
		return
	}
	targetID, err := pathID(r, "user_id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Role domain.Role `json:"role"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	member, err := s.svc.ChangeMemberRole(r.Context(), userID(r.Context()), teamID, targetID, req.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMemberResponse(member))
}

func (s *Server) handleTeamStats(w http.ResponseWriter, r *http.Request) {
	teamID, err := pathID(r, "team_id")
	if err != nil {
		writeError(w, err)
		return
	}

	stats, err := s.svc.TeamStats(r.Context(), userID(r.Context()), teamID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
