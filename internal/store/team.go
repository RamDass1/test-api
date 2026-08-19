package store

import (
	"context"
	"fmt"

	"github.com/RamDass1/test-api/internal/domain"
)

func (s *Store) CreateTeam(ctx context.Context, name string, createdBy int64) (domain.Team, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Team{}, fmt.Errorf("begin transaction: %w", err)
	}

	const insertTeam = `INSERT INTO teams (name, created_by) VALUES (?, ?)`
	res, err := tx.ExecContext(ctx, insertTeam, name, createdBy)
	if err != nil {
		_ = tx.Rollback()
		return domain.Team{}, classify(err, "insert team")
	}
	id, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return domain.Team{}, classify(err, "insert team id")
	}

	const insertOwner = `INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insertOwner, id, createdBy, domain.RoleOwner); err != nil {
		_ = tx.Rollback()
		return domain.Team{}, classify(err, "insert team owner")
	}

	const read = `SELECT id, name, created_by, created_at FROM teams WHERE id = ?`
	var t domain.Team
	if err := tx.QueryRowContext(ctx, read, id).Scan(&t.ID, &t.Name, &t.CreatedBy, &t.CreatedAt); err != nil {
		_ = tx.Rollback()
		return domain.Team{}, noRows(err, "select team")
	}

	if err := tx.Commit(); err != nil {
		return domain.Team{}, fmt.Errorf("commit: %w", err)
	}
	t.Role = domain.RoleOwner
	return t, nil
}

func (s *Store) AddMember(ctx context.Context, teamID, userID int64, role domain.Role) error {
	const q = `INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`
	if _, err := s.db.ExecContext(ctx, q, teamID, userID, role); err != nil {
		return classify(err, "insert team member")
	}
	return nil
}

func (s *Store) UpdateMemberRole(ctx context.Context, teamID, userID int64, role domain.Role) error {
	const q = `UPDATE team_members SET role = ? WHERE team_id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, q, role, teamID, userID)
	if err != nil {
		return classify(err, "update member role")
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	if affected == 0 {
		if _, err := s.Membership(ctx, teamID, userID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Membership(ctx context.Context, teamID, userID int64) (domain.TeamMember, error) {
	const q = `
		SELECT m.team_id, m.user_id, m.role, u.email, u.name
		FROM team_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.team_id = ? AND m.user_id = ?`
	var m domain.TeamMember
	err := s.db.QueryRowContext(ctx, q, teamID, userID).Scan(&m.TeamID, &m.UserID, &m.Role, &m.Email, &m.Name)
	if err != nil {
		return domain.TeamMember{}, noRows(err, "select membership")
	}
	return m, nil
}

func (s *Store) TeamsForUser(ctx context.Context, userID int64) ([]domain.Team, error) {
	const q = `
		SELECT t.id, t.name, t.created_by, t.created_at, m.role
		FROM teams t
		JOIN team_members m ON m.team_id = t.id
		WHERE m.user_id = ?
		ORDER BY t.id`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("select teams: %w", err)
	}
	defer rows.Close()

	teams := make([]domain.Team, 0)
	for rows.Next() {
		var t domain.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedBy, &t.CreatedAt, &t.Role); err != nil {
			return nil, fmt.Errorf("scan team: %w", err)
		}
		teams = append(teams, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teams: %w", err)
	}
	return teams, nil
}
