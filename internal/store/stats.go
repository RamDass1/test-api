package store

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/RamDass1/test-api/internal/domain"
)

const statsQuery = `
WITH scoped_tasks AS (
    SELECT id, status, assignee_id, created_at, closed_at
    FROM tasks
    WHERE team_id = ?
),
status_counts AS (
    SELECT CAST(status AS CHAR) AS status, COUNT(*) AS cnt
    FROM scoped_tasks
    GROUP BY status
),
closed_last_30d AS (
    SELECT t.assignee_id AS user_id, u.name AS name, COUNT(*) AS closed_tasks
    FROM scoped_tasks t
    JOIN users u ON u.id = t.assignee_id
    WHERE t.closed_at IS NOT NULL
      AND t.closed_at >= NOW(6) - INTERVAL 30 DAY
    GROUP BY t.assignee_id, u.name
    ORDER BY closed_tasks DESC, u.name ASC
    LIMIT 3
)
SELECT
    (SELECT COALESCE(JSON_OBJECTAGG(status, cnt), JSON_OBJECT())
       FROM status_counts) AS tasks_by_status,
    (SELECT COALESCE(
                JSON_ARRAYAGG(JSON_OBJECT('user_id', user_id, 'name', name, 'closed_tasks', closed_tasks)),
                JSON_ARRAY())
       FROM closed_last_30d) AS top_assignees,
    (SELECT AVG(TIMESTAMPDIFF(SECOND, created_at, closed_at))
       FROM scoped_tasks
      WHERE closed_at IS NOT NULL) AS avg_close_seconds,
    (SELECT COUNT(*)
       FROM task_comments c
       JOIN scoped_tasks t ON t.id = c.task_id) AS comments_total`

func (s *Store) TeamStats(ctx context.Context, teamID int64) (domain.TeamStats, error) {
	var (
		stats        = domain.TeamStats{TeamID: teamID}
		byStatusRaw  []byte
		assigneesRaw []byte
		avgSeconds   sql.NullFloat64
	)

	err := s.db.QueryRowContext(ctx, statsQuery, teamID).Scan(
		&byStatusRaw,
		&assigneesRaw,
		&avgSeconds,
		&stats.CommentsTotal,
	)
	if err != nil {
		return domain.TeamStats{}, fmt.Errorf("select team stats: %w", err)
	}

	if err := json.Unmarshal(byStatusRaw, &stats.TasksByStatus); err != nil {
		return domain.TeamStats{}, fmt.Errorf("decode tasks_by_status: %w", err)
	}
	if err := json.Unmarshal(assigneesRaw, &stats.TopAssignees30d); err != nil {
		return domain.TeamStats{}, fmt.Errorf("decode top_assignees: %w", err)
	}
	slices.SortStableFunc(stats.TopAssignees30d, func(a, b domain.AssigneeStat) int {
		if c := cmp.Compare(b.ClosedTasks, a.ClosedTasks); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})

	if avgSeconds.Valid {
		stats.AvgCloseSeconds = &avgSeconds.Float64
	}
	if stats.TasksByStatus == nil {
		stats.TasksByStatus = map[domain.Status]int64{}
	}
	if stats.TopAssignees30d == nil {
		stats.TopAssignees30d = []domain.AssigneeStat{}
	}
	return stats, nil
}
