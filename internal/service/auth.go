package service

import (
	"context"
	"errors"
	"time"

	"github.com/RamDass1/test-api/internal/domain"
)

type Session struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      domain.User `json:"user"`
}

func (s *Service) Register(ctx context.Context, email, name, password string) (domain.User, error) {
	email, err := domain.NormalizeEmail(email)
	if err != nil {
		return domain.User{}, err
	}
	name, err = domain.NormalizeName(name)
	if err != nil {
		return domain.User{}, err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return domain.User{}, err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, domain.Internal(err, "hash password")
	}

	user, err := s.db.CreateUser(ctx, email, name, hash)
	if errors.Is(err, domain.ErrAlreadyExists) {
		return domain.User{}, domain.Conflict("a user with this email already exists")
	}
	if err != nil {
		return domain.User{}, domain.Internal(err, "create user")
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	email, err := domain.NormalizeEmail(email)
	if err != nil {
		return Session{}, domain.Unauthorized("invalid email or password")
	}

	creds, err := s.db.CredentialsByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return Session{}, domain.Unauthorized("invalid email or password")
	}
	if err != nil {
		return Session{}, domain.Internal(err, "load credentials")
	}
	if !s.hasher.Verify(creds.PasswordHash, password) {
		return Session{}, domain.Unauthorized("invalid email or password")
	}

	token, exp, err := s.tokens.Issue(creds.User.ID)
	if err != nil {
		return Session{}, domain.Internal(err, "issue token")
	}
	return Session{Token: token, ExpiresAt: exp, User: creds.User}, nil
}
