package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/RamDass1/test-api/internal/domain"
	"github.com/RamDass1/test-api/internal/store"
)

const day = 24 * 60 * 60

func TestTeamStats(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set TEST_MYSQL_DSN")
	}

	ctx := context.Background()
	st, db := newTestSchema(t, dsn)
	users := seedUsers(t, db, "ann", "bob", "outsider")

	teamID := seedTeam(t, db, "platform", users["ann"])
	addMember(t, db, teamID, users["bob"], domain.RoleMember)

	first := seedTask(t, db, teamID, users["ann"], users["ann"], domain.StatusDone, 2*day, 1*day)
	seedTask(t, db, teamID, users["ann"], users["bob"], domain.StatusDone, 2*day, 1*day)
	seedTask(t, db, teamID, users["ann"], users["ann"], domain.StatusDone, 70*day, 60*day)
	seedTask(t, db, teamID, users["ann"], 0, domain.StatusTodo, 1*day, 0)

	if _, err := db.Exec(`INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?), (?, ?, ?)`,
		first, users["ann"], "a", first, users["bob"], "b"); err != nil {
		t.Fatalf("insert comments: %v", err)
	}

	other := seedTeam(t, db, "outsiders", users["outsider"])
	otherTask := seedTask(t, db, other, users["outsider"], users["outsider"], domain.StatusDone, 2*day, 1*day)
	if _, err := db.Exec(`INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`,
		otherTask, users["outsider"], "hidden"); err != nil {
		t.Fatalf("insert outsider comment: %v", err)
	}

	stats, err := st.TeamStats(ctx, teamID)
	if err != nil {
		t.Fatalf("TeamStats: %v", err)
	}
	if stats.TeamID != teamID {
		t.Errorf("team_id = %d, want %d", stats.TeamID, teamID)
	}
	if got, want := stats.TasksByStatus, map[domain.Status]int64{domain.StatusDone: 3, domain.StatusTodo: 1}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("tasks_by_status = %v, want %v", got, want)
	}

	wantTop := []domain.AssigneeStat{
		{UserID: users["ann"], Name: "ann", ClosedTasks: 1},
		{UserID: users["bob"], Name: "bob", ClosedTasks: 1},
	}
	if fmt.Sprint(stats.TopAssignees30d) != fmt.Sprint(wantTop) {
		t.Errorf("top_assignees_30d = %+v, want %+v", stats.TopAssignees30d, wantTop)
	}

	const wantAvg = float64(1*day+1*day+10*day) / 3
	if stats.AvgCloseSeconds == nil || math.Abs(*stats.AvgCloseSeconds-wantAvg) > 1 {
		t.Errorf("avg_close_seconds = %v, want %.0f", stats.AvgCloseSeconds, wantAvg)
	}
	if stats.CommentsTotal != 2 {
		t.Errorf("comments_total = %d, want 2", stats.CommentsTotal)
	}
}

func newTestSchema(t *testing.T, dsn string) (*store.Store, *sql.DB) {
	t.Helper()

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse TEST_MYSQL_DSN: %v", err)
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC

	schema := fmt.Sprintf("test_api_test_%d", time.Now().UnixNano())
	admin := *cfg
	admin.DBName = ""
	adminDB, err := sql.Open("mysql", admin.FormatDSN())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := adminDB.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		adminDB.Close()
		t.Fatalf("create schema (need CREATE privilege, use root): %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.Exec("DROP DATABASE IF EXISTS `" + schema + "`")
		adminDB.Close()
	})

	cfg.DBName = schema
	ctx := context.Background()
	if err := store.Migrate(ctx, cfg.FormatDSN()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	st, err := store.Open(ctx, cfg.FormatDSN())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st, db
}

func seedUsers(t *testing.T, db *sql.DB, names ...string) map[string]int64 {
	t.Helper()
	ids := make(map[string]int64, len(names))
	for _, name := range names {
		res, err := db.Exec(`INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`, name+"@example.com", "x", name)
		if err != nil {
			t.Fatalf("insert user %s: %v", name, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("user id %s: %v", name, err)
		}
		ids[name] = id
	}
	return ids
}

func seedTeam(t *testing.T, db *sql.DB, name string, ownerID int64) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO teams (name, created_by) VALUES (?, ?)`, name, ownerID)
	if err != nil {
		t.Fatalf("insert team %s: %v", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("team id %s: %v", name, err)
	}
	addMember(t, db, id, ownerID, domain.RoleOwner)
	return id
}

func addMember(t *testing.T, db *sql.DB, teamID, userID int64, role domain.Role) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`, teamID, userID, role); err != nil {
		t.Fatalf("add member: %v", err)
	}
}

func seedTask(t *testing.T, db *sql.DB, teamID, createdBy, assigneeID int64, status domain.Status, createdAgo, closedAgo int) int64 {
	t.Helper()
	var assignee any
	if assigneeID > 0 {
		assignee = assigneeID
	}
	closedAt := "NULL"
	args := []any{teamID, status, createdBy, assignee, createdAgo}
	if closedAgo > 0 {
		closedAt = "NOW(6) - INTERVAL ? SECOND"
		args = append(args, closedAgo)
	}
	res, err := db.Exec(`
		INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at)
		VALUES (?, 'task', '', ?, ?, ?, NOW(6) - INTERVAL ? SECOND, NOW(6), `+closedAt+`)`, args...)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("task id: %v", err)
	}
	return id
}
