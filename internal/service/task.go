package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/RamDass1/test-api/internal/domain"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type NewTaskRequest struct {
	TeamID      int64
	Title       string
	Description string
	Status      domain.Status
	AssigneeID  *int64
}

func (s *Service) CreateTask(ctx context.Context, actorID int64, req NewTaskRequest) (domain.Task, error) {
	if req.TeamID <= 0 {
		return domain.Task{}, domain.Invalid("team_id is required")
	}
	if req.Status == "" {
		req.Status = domain.StatusTodo
	}
	if err := domain.ValidateNewTask(req.Title, req.Description, req.Status); err != nil {
		return domain.Task{}, err
	}
	if _, err := membership(ctx, s.db, req.TeamID, actorID); err != nil {
		return domain.Task{}, err
	}
	if err := assigneeInTeam(ctx, s.db, req.TeamID, req.AssigneeID); err != nil {
		return domain.Task{}, err
	}

	task, err := s.db.CreateTask(ctx, domain.NewTask{
		TeamID:      req.TeamID,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Status:      req.Status,
		CreatedBy:   actorID,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		return domain.Task{}, domain.Internal(err, "create task")
	}
	s.cache.Invalidate(ctx, task.TeamID)
	return task, nil
}

func (s *Service) UpdateTask(ctx context.Context, actorID, taskID int64, patch domain.TaskPatch) (domain.Task, error) {
	if err := patch.Validate(); err != nil {
		return domain.Task{}, err
	}

	task, actor, err := taskForMember(ctx, s.db, taskID, actorID)
	if err != nil {
		return domain.Task{}, err
	}
	if patch.Version != nil && *patch.Version != task.Version {
		return domain.Task{}, domain.Conflict("task was modified by someone else (current version %d)", task.Version)
	}

	changes := domain.Diff(task, patch)
	if err := domain.AuthorizeTaskUpdate(actor, task, changes); err != nil {
		return domain.Task{}, err
	}
	if len(changes) == 0 {
		return task, nil
	}

	next := domain.Apply(task, changes)
	if err := assigneeInTeam(ctx, s.db, task.TeamID, next.AssigneeID); err != nil {
		return domain.Task{}, err
	}

	saved, err := s.db.UpdateTask(ctx, task.ID, task.Version, domain.TaskUpdate{
		Title:       next.Title,
		Description: next.Description,
		Status:      next.Status,
		AssigneeID:  next.AssigneeID,
		ClosedAt:    domain.CloseTimestamp(task, next.Status, time.Now()),
	}, actorID, changes)
	if errors.Is(err, domain.ErrVersionConflict) {
		return domain.Task{}, domain.Conflict("task was modified by someone else, retry with current version")
	}
	if err != nil {
		return domain.Task{}, domain.Internal(err, "update task")
	}
	s.cache.Invalidate(ctx, saved.TeamID)
	return saved, nil
}

func (s *Service) ListTasks(ctx context.Context, actorID int64, filter domain.TaskFilter) (domain.TaskPage, error) {
	if filter.TeamID <= 0 {
		return domain.TaskPage{}, domain.Invalid("team_id is required")
	}
	if filter.Status != nil && !filter.Status.Valid() {
		return domain.TaskPage{}, domain.Invalid("unknown status %q", *filter.Status)
	}
	filter.Limit = normalizeLimit(filter.Limit)
	filter.Offset = normalizeOffset(filter.Offset)

	if _, err := membership(ctx, s.db, filter.TeamID, actorID); err != nil {
		return domain.TaskPage{}, err
	}

	if page, ok := s.cache.Get(ctx, filter); ok {
		return page, nil
	}

	tasks, err := s.db.ListTasks(ctx, filter)
	if err != nil {
		return domain.TaskPage{}, domain.Internal(err, "list tasks")
	}
	page := domain.TaskPage{Items: tasks, Limit: filter.Limit, Offset: filter.Offset}
	s.cache.Set(ctx, filter, page)
	return page, nil
}

func (s *Service) TaskHistory(ctx context.Context, actorID, taskID int64) ([]domain.HistoryEntry, error) {
	if _, _, err := taskForMember(ctx, s.db, taskID, actorID); err != nil {
		return nil, err
	}
	entries, err := s.db.TaskHistory(ctx, taskID)
	if err != nil {
		return nil, domain.Internal(err, "list task history")
	}
	return entries, nil
}

func (s *Service) AddComment(ctx context.Context, actorID, taskID int64, content string) (domain.Comment, error) {
	if err := domain.ValidateComment(content); err != nil {
		return domain.Comment{}, err
	}
	if _, _, err := taskForMember(ctx, s.db, taskID, actorID); err != nil {
		return domain.Comment{}, err
	}

	comment, err := s.db.AddComment(ctx, taskID, actorID, strings.TrimSpace(content))
	if err != nil {
		return domain.Comment{}, domain.Internal(err, "add comment")
	}
	return comment, nil
}

func (s *Service) ListComments(ctx context.Context, actorID, taskID int64) ([]domain.Comment, error) {
	if _, _, err := taskForMember(ctx, s.db, taskID, actorID); err != nil {
		return nil, err
	}
	comments, err := s.db.TaskComments(ctx, taskID)
	if err != nil {
		return nil, domain.Internal(err, "list comments")
	}
	return comments, nil
}

func taskForMember(ctx context.Context, db DB, taskID, actorID int64) (domain.Task, domain.Actor, error) {
	task, err := db.TaskByID(ctx, taskID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Task{}, domain.Actor{}, domain.NotFound("task %d not found", taskID)
	}
	if err != nil {
		return domain.Task{}, domain.Actor{}, domain.Internal(err, "load task")
	}
	actor, err := membership(ctx, db, task.TeamID, actorID)
	if err != nil {
		return domain.Task{}, domain.Actor{}, domain.NotFound("task %d not found", taskID)
	}
	return task, actor, nil
}

func assigneeInTeam(ctx context.Context, db DB, teamID int64, assigneeID *int64) error {
	if assigneeID == nil {
		return nil
	}
	if _, err := db.Membership(ctx, teamID, *assigneeID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.Invalid("assignee %d is not a member of team %d", *assigneeID, teamID)
		}
		return domain.Internal(err, "verify assignee membership")
	}
	return nil
}

func normalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultLimit
	case limit > MaxLimit:
		return MaxLimit
	default:
		return limit
	}
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
