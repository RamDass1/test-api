package httpapi

import (
	"context"
	"net/http"

	"github.com/RamDass1/test-api/internal/service"
)

type Server struct {
	svc     *service.Service
	tokens  TokenParser
	limiter *ipRateLimiter
}

func New(svc *service.Service, tokens TokenParser) *Server {
	return &Server{
		svc:     svc,
		tokens:  tokens,
		limiter: newIPRateLimiter(20, 40),
	}
}

func (s *Server) Collect(ctx context.Context) {
	if s.limiter == nil {
		return
	}
	s.limiter.collect(ctx)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/login", s.handleLogin)

	mux.Handle("POST /api/v1/teams", s.authenticate(http.HandlerFunc(s.handleCreateTeam)))
	mux.Handle("GET /api/v1/teams", s.authenticate(http.HandlerFunc(s.handleListTeams)))
	mux.Handle("POST /api/v1/teams/{team_id}/invite", s.authenticate(http.HandlerFunc(s.handleInvite)))
	mux.Handle("PATCH /api/v1/teams/{team_id}/members/{user_id}", s.authenticate(http.HandlerFunc(s.handleChangeRole)))
	mux.Handle("GET /api/v1/teams/{team_id}/stats", s.authenticate(http.HandlerFunc(s.handleTeamStats)))

	mux.Handle("POST /api/v1/tasks", s.authenticate(http.HandlerFunc(s.handleCreateTask)))
	mux.Handle("GET /api/v1/tasks", s.authenticate(http.HandlerFunc(s.handleListTasks)))
	mux.Handle("PUT /api/v1/tasks/{task_id}", s.authenticate(http.HandlerFunc(s.handleUpdateTask)))
	mux.Handle("GET /api/v1/tasks/{task_id}/history", s.authenticate(http.HandlerFunc(s.handleTaskHistory)))
	mux.Handle("POST /api/v1/tasks/{task_id}/comments", s.authenticate(http.HandlerFunc(s.handleAddComment)))
	mux.Handle("GET /api/v1/tasks/{task_id}/comments", s.authenticate(http.HandlerFunc(s.handleListComments)))

	return withRequestID(requestLogger(s.rateLimit(mux)))
}