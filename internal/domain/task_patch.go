package domain

import (
	"strings"
	"unicode/utf8"
)

type TaskPatch struct {
	Title       Optional[string] `json:"title"`
	Description Optional[string] `json:"description"`
	Status      Optional[Status] `json:"status"`
	AssigneeID  Optional[int64]  `json:"assignee_id"`
	Version     *int64           `json:"version"`
}

func (p TaskPatch) Empty() bool {
	return !p.Title.Set && !p.Description.Set && !p.Status.Set && !p.AssigneeID.Set
}

func (p TaskPatch) Validate() error {
	if p.Title.Set {
		if !p.Title.Valid {
			return Invalid("title cannot be null")
		}
		if err := validateTitle(p.Title.Value); err != nil {
			return err
		}
	}
	if p.Description.Set {
		if !p.Description.Valid {
			return Invalid("description cannot be null")
		}
		if utf8.RuneCountInString(p.Description.Value) > maxDescriptionLen {
			return Invalid("description must be at most %d characters", maxDescriptionLen)
		}
	}
	if p.Status.Set {
		if !p.Status.Valid {
			return Invalid("status cannot be null")
		}
		if !p.Status.Value.Valid() {
			return Invalid("unknown status %q", p.Status.Value)
		}
	}
	if p.AssigneeID.Set && p.AssigneeID.Valid && p.AssigneeID.Value <= 0 {
		return Invalid("assignee_id must be a positive integer")
	}
	if p.Version != nil && *p.Version <= 0 {
		return Invalid("version must be a positive integer")
	}
	return nil
}

func Diff(task Task, patch TaskPatch) map[string]FieldChange {
	changes := make(map[string]FieldChange)

	if patch.Title.Set {
		next := strings.TrimSpace(patch.Title.Value)
		if next != task.Title {
			changes[FieldTitle] = FieldChange{From: task.Title, To: next}
		}
	}
	if patch.Description.Set && patch.Description.Value != task.Description {
		changes[FieldDescription] = FieldChange{From: task.Description, To: patch.Description.Value}
	}
	if patch.Status.Set && patch.Status.Value != task.Status {
		changes[FieldStatus] = FieldChange{From: task.Status, To: patch.Status.Value}
	}
	if patch.AssigneeID.Set {
		var next *int64
		if patch.AssigneeID.Valid {
			v := patch.AssigneeID.Value
			next = &v
		}
		if !sameAssignee(task.AssigneeID, next) {
			changes[FieldAssignee] = FieldChange{From: task.AssigneeID, To: next}
		}
	}

	return changes
}

func Apply(task Task, changes map[string]FieldChange) Task {
	if c, ok := changes[FieldTitle]; ok {
		task.Title, _ = c.To.(string)
	}
	if c, ok := changes[FieldDescription]; ok {
		task.Description, _ = c.To.(string)
	}
	if c, ok := changes[FieldStatus]; ok {
		task.Status, _ = c.To.(Status)
	}
	if c, ok := changes[FieldAssignee]; ok {
		task.AssigneeID, _ = c.To.(*int64)
	}
	return task
}

func sameAssignee(a, b *int64) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}