package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/RamDass1/test-api/internal/domain"
)

const taskColumns = `id, team_id, title, description, status, created_by, assignee_id,
	created_at, updated_at, closed_at, version`

func (s *Store) CreateTask(ctx context.Context, in domain.NewTask) (domain.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Task{}, fmt.Errorf("begin transaction: %w", err)
	}

	var closedAt *time.Time
	if in.Status.Terminal() {
		now := time.Now().UTC()
		closedAt = &now
	}

	const insertTask = `
		INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id, closed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, insertTask,
		in.TeamID, in.Title, in.Description, in.Status, in.CreatedBy, in.AssigneeID, closedAt)
	if err != nil {
		_ = tx.Rollback()
		return domain.Task{}, classify(err, "insert task")
	}
	id, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return domain.Task{}, classify(err, "insert task id")
	}

	task, err := scanTask(tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id))
	if err != nil {
		_ = tx.Rollback()
		return domain.Task{}, err
	}

	if err := insertHistory(ctx, tx, task.ID, in.CreatedBy, domain.CreationChanges(task)); err != nil {
		_ = tx.Rollback()
		return domain.Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Task{}, fmt.Errorf("commit: %w", err)
	}
	return task, nil
}

func (s *Store) TaskByID(ctx context.Context, id int64) (domain.Task, error) {
	return scanTask(s.db.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id))
}

func (s *Store) UpdateTask(ctx context.Context, id, expectedVersion int64, upd domain.TaskUpdate, changedBy int64, changes map[string]domain.FieldChange) (domain.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Task{}, fmt.Errorf("begin transaction: %w", err)
	}

	const updateTask = `
		UPDATE tasks
		SET title = ?, description = ?, status = ?, assignee_id = ?, closed_at = ?,
		    updated_at = NOW(6), version = version + 1
		WHERE id = ? AND version = ?`
	res, err := tx.ExecContext(ctx, updateTask,
		upd.Title, upd.Description, upd.Status, upd.AssigneeID, upd.ClosedAt, id, expectedVersion)
	if err != nil {
		_ = tx.Rollback()
		return domain.Task{}, classify(err, "update task")
	}
	affected, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return domain.Task{}, fmt.Errorf("update task: %w", err)
	}
	if affected == 0 {
		_ = tx.Rollback()
		return domain.Task{}, fmt.Errorf("update task %d: %w", id, domain.ErrVersionConflict)
	}

	if err := insertHistory(ctx, tx, id, changedBy, changes); err != nil {
		_ = tx.Rollback()
		return domain.Task{}, err
	}

	task, err := scanTask(tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id))
	if err != nil {
		_ = tx.Rollback()
		return domain.Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Task{}, fmt.Errorf("commit: %w", err)
	}
	return task, nil
}

func (s *Store) ListTasks(ctx context.Context, f domain.TaskFilter) ([]domain.Task, error) {
	where := []string{"team_id = ?"}
	args := []any{f.TeamID}

	if f.Status != nil {
		where = append(where, "status = ?")
		args = append(args, *f.Status)
	}
	if f.AssigneeID != nil {
		where = append(where, "assignee_id = ?")
		args = append(args, *f.AssigneeID)
	}

	query := `SELECT ` + taskColumns + ` FROM tasks WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Limit, f.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		var t domain.Task
		if err := rows.Scan(&t.ID, &t.TeamID, &t.Title, &t.Description, &t.Status, &t.CreatedBy,
			&t.AssigneeID, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt, &t.Version); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func (s *Store) TaskHistory(ctx context.Context, taskID int64) ([]domain.HistoryEntry, error) {
	const q = `
		SELECT id, task_id, changed_by, changes, created_at
		FROM task_history
		WHERE task_id = ?
		ORDER BY id DESC`

	rows, err := s.db.QueryContext(ctx, q, taskID)
	if err != nil {
		return nil, fmt.Errorf("select task history: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.HistoryEntry, 0)
	for rows.Next() {
		var (
			e   domain.HistoryEntry
			raw []byte
		)
		if err := rows.Scan(&e.ID, &e.TaskID, &e.ChangedBy, &raw, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan history entry: %w", err)
		}
		if err := json.Unmarshal(raw, &e.Changes); err != nil {
			return nil, fmt.Errorf("decode history changes: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task history: %w", err)
	}
	return entries, nil
}

func scanTask(row *sql.Row) (domain.Task, error) {
	var t domain.Task
	err := row.Scan(&t.ID, &t.TeamID, &t.Title, &t.Description, &t.Status, &t.CreatedBy,
		&t.AssigneeID, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt, &t.Version)
	if err != nil {
		return domain.Task{}, noRows(err, "scan task")
	}
	return t, nil
}

func insertHistory(ctx context.Context, tx *sql.Tx, taskID, changedBy int64, changes map[string]domain.FieldChange) error {
	payload, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("encode task changes: %w", err)
	}
	const q = `INSERT INTO task_history (task_id, changed_by, changes) VALUES (?, ?, ?)`
	if _, err := tx.ExecContext(ctx, q, taskID, changedBy, payload); err != nil {
		return classify(err, "insert task history")
	}
	return nil
}