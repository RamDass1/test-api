package store

import (
	"context"
	"fmt"

	"github.com/RamDass1/test-api/internal/domain"
)

func (s *Store) AddComment(ctx context.Context, taskID, userID int64, content string) (domain.Comment, error) {
	const insert = `INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`
	res, err := s.db.ExecContext(ctx, insert, taskID, userID, content)
	if err != nil {
		return domain.Comment{}, classify(err, "insert comment")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Comment{}, classify(err, "insert comment id")
	}

	const read = `SELECT id, task_id, user_id, content, created_at FROM task_comments WHERE id = ?`
	var c domain.Comment
	if err := s.db.QueryRowContext(ctx, read, id).
		Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content, &c.CreatedAt); err != nil {
		return domain.Comment{}, noRows(err, "select comment")
	}
	return c, nil
}

func (s *Store) TaskComments(ctx context.Context, taskID int64) ([]domain.Comment, error) {
	const q = `
		SELECT id, task_id, user_id, content, created_at
		FROM task_comments
		WHERE task_id = ?
		ORDER BY id DESC`

	rows, err := s.db.QueryContext(ctx, q, taskID)
	if err != nil {
		return nil, fmt.Errorf("select comments: %w", err)
	}
	defer rows.Close()

	comments := make([]domain.Comment, 0)
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comments: %w", err)
	}
	return comments, nil
}
