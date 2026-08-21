package domain

import "time"

const (
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldStatus      = "status"
	FieldAssignee    = "assignee_id"
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone, StatusCancelled:
		return true
	}
	return false
}

func (s Status) Terminal() bool {
	return s == StatusDone || s == StatusCancelled
}

type Task struct {
	ID          int64      `json:"id"`
	TeamID      int64      `json:"team_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	CreatedBy   int64      `json:"created_by"`
	AssigneeID  *int64     `json:"assignee_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	Version     int64      `json:"version"`
}

type NewTask struct {
	TeamID      int64
	Title       string
	Description string
	Status      Status
	CreatedBy   int64
	AssigneeID  *int64
}

type TaskUpdate struct {
	Title       string
	Description string
	Status      Status
	AssigneeID  *int64
	ClosedAt    *time.Time
}

type FieldChange struct {
	From any `json:"from"`
	To   any `json:"to"`
}

type HistoryEntry struct {
	ID        int64                  `json:"id"`
	TaskID    int64                  `json:"task_id"`
	ChangedBy int64                  `json:"changed_by"`
	Changes   map[string]FieldChange `json:"changes"`
	CreatedAt time.Time              `json:"created_at"`
}

type TaskFilter struct {
	TeamID     int64
	Status     *Status
	AssigneeID *int64
	Limit      int
	Offset     int
	Cursor     *int64
}

type TaskPage struct {
	Items      []Task `json:"items"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	NextCursor *int64 `json:"next_cursor,omitempty"`
}

func CloseTimestamp(current Task, next Status, now time.Time) *time.Time {
	switch {
	case !next.Terminal():
		return nil
	case current.ClosedAt != nil:
		return current.ClosedAt
	default:
		stamped := now.UTC()
		return &stamped
	}
}

func CreationChanges(task Task) map[string]FieldChange {
	changes := map[string]FieldChange{
		FieldTitle:       {From: nil, To: task.Title},
		FieldStatus:      {From: nil, To: task.Status},
		FieldDescription: {From: nil, To: task.Description},
	}
	if task.AssigneeID != nil {
		changes[FieldAssignee] = FieldChange{From: nil, To: task.AssigneeID}
	}
	return changes
}
