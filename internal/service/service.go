package service

import (
	"context"
	"time"

	"github.com/RamDass1/test-api/internal/domain"
)

type DB interface {
	InTx(ctx context.Context, fn func(DB) error) error

	CreateUser(ctx context.Context, email, name, passwordHash string) (domain.User, error)
	UserByID(ctx context.Context, id int64) (domain.User, error)
	UserByEmail(ctx context.Context, email string) (domain.User, error)
	CredentialsByEmail(ctx context.Context, email string) (domain.Credentials, error)

	CreateTeam(ctx context.Context, name string, createdBy int64) (domain.Team, error)
	AddMember(ctx context.Context, teamID, userID int64, role domain.Role) error
	UpdateMemberRole(ctx context.Context, teamID, userID int64, role domain.Role) error
	Membership(ctx context.Context, teamID, userID int64) (domain.TeamMember, error)
	TeamsForUser(ctx context.Context, userID int64) ([]domain.Team, error)
}

type Hasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) bool
}

type Tokens interface {
	Issue(userID int64) (string, time.Time, error)
}

type Service struct {
	db     DB
	hasher Hasher
	tokens Tokens
}

func New(db DB, hasher Hasher, tokens Tokens) *Service {
	return &Service{db: db, hasher: hasher, tokens: tokens}
}
