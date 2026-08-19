package service

import (
	"context"
	"time"

	"github.com/RamDass1/test-api/internal/domain"
)

type DB interface {
	CreateUser(ctx context.Context, email, name, passwordHash string) (domain.User, error)
	UserByID(ctx context.Context, id int64) (domain.User, error)
	UserByEmail(ctx context.Context, email string) (domain.User, error)
	CredentialsByEmail(ctx context.Context, email string) (domain.Credentials, error)

	CreateTeam(ctx context.Context, name string, createdBy int64) (domain.Team, error)
	AddMember(ctx context.Context, teamID, userID int64, role domain.Role) error
	UpdateMemberRole(ctx context.Context, teamID, userID int64, role domain.Role) error
	Membership(ctx context.Context, teamID, userID int64) (domain.TeamMember, error)
	TeamsForUser(ctx context.Context, userID int64) ([]domain.Team, error)

	CreateTask(ctx context.Context, in domain.NewTask) (domain.Task, error)
	TaskByID(ctx context.Context, id int64) (domain.Task, error)
	UpdateTask(ctx context.Context, id, expectedVersion int64, upd domain.TaskUpdate, changedBy int64, changes map[string]domain.FieldChange) (domain.Task, error)
	ListTasks(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error)
	TaskHistory(ctx context.Context, taskID int64) ([]domain.HistoryEntry, error)
}

type Cache interface {
	Get(ctx context.Context, filter domain.TaskFilter) (domain.TaskPage, bool)
	Set(ctx context.Context, filter domain.TaskFilter, page domain.TaskPage)
	Invalidate(ctx context.Context, teamID int64)
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
	cache  Cache
	hasher Hasher
	tokens Tokens
}

func New(db DB, cache Cache, hasher Hasher, tokens Tokens) *Service {
	return &Service{db: db, cache: cache, hasher: hasher, tokens: tokens}
}