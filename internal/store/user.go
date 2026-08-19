package store

import (
	"context"

	"github.com/RamDass1/test-api/internal/domain"
)

func (s *Store) CreateUser(ctx context.Context, email, name, passwordHash string) (domain.User, error) {
	const q = `INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`
	res, err := s.db.ExecContext(ctx, q, email, passwordHash, name)
	if err != nil {
		return domain.User{}, classify(err, "insert user")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.User{}, classify(err, "insert user id")
	}
	return s.UserByID(ctx, id)
}

func (s *Store) UserByID(ctx context.Context, id int64) (domain.User, error) {
	const q = `SELECT id, email, name, created_at FROM users WHERE id = ?`
	var u domain.User
	err := s.db.QueryRowContext(ctx, q, id).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
	if err != nil {
		return domain.User{}, noRows(err, "select user")
	}
	return u, nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	const q = `SELECT id, email, name, created_at FROM users WHERE email = ?`
	var u domain.User
	err := s.db.QueryRowContext(ctx, q, email).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
	if err != nil {
		return domain.User{}, noRows(err, "select user by email")
	}
	return u, nil
}

func (s *Store) CredentialsByEmail(ctx context.Context, email string) (domain.Credentials, error) {
	const q = `SELECT id, email, name, created_at, password_hash FROM users WHERE email = ?`
	var c domain.Credentials
	err := s.db.QueryRowContext(ctx, q, email).
		Scan(&c.User.ID, &c.User.Email, &c.User.Name, &c.User.CreatedAt, &c.PasswordHash)
	if err != nil {
		return domain.Credentials{}, noRows(err, "select credentials")
	}
	return c, nil
}
