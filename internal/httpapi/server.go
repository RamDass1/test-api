package httpapi

import (
	"net/http"

	"github.com/RamDass1/test-api/internal/service"
)

type Server struct {
	svc    *service.Service
	tokens TokenParser
}

func New(svc *service.Service, tokens TokenParser) *Server {
	return &Server{svc: svc, tokens: tokens}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/login", s.handleLogin)

	mux.Handle("POST /api/v1/teams", s.authenticate(http.HandlerFunc(s.handleCreateTeam)))
	mux.Handle("GET /api/v1/teams", s.authenticate(http.HandlerFunc(s.handleListTeams)))
	mux.Handle("POST /api/v1/teams/{team_id}/invite", s.authenticate(http.HandlerFunc(s.handleInvite)))
	mux.Handle("PATCH /api/v1/teams/{team_id}/members/{user_id}", s.authenticate(http.HandlerFunc(s.handleChangeRole)))
	return mux
}
