package service

import (
	"context"
	"time"

	"github.com/RamDass1/test-api/internal/domain"
)

type DB interface {
	CreateUser(ctx context.Context, email, name, passwordHash string) (domain.User, error)
	CredentialsByEmail(ctx context.Context, email string) (domain.Credentials, error)
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
