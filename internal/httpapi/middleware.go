package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/RamDass1/test-api/internal/domain"
)

type TokenParser interface {
	Parse(token string) (int64, error)
}

type contextKey int

const userIDKey contextKey = iota

func userID(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
			writeError(w, domain.Unauthorized("a bearer token is required"))
			return
		}
		id, err := s.tokens.Parse(token)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			writeError(w, domain.Unauthorized("the token is invalid or has expired"))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, id)))
	})
}

func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}
