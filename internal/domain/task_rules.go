package domain

import (
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	maxTitleLen       = 255
	maxDescriptionLen = 10000
)

var (
	creatorFields  = []string{FieldTitle, FieldDescription, FieldStatus, FieldAssignee}
	assigneeFields = []string{FieldStatus}
)

func validateTitle(title string) error {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return Invalid("title must not be empty")
	}
	if utf8.RuneCountInString(trimmed) > maxTitleLen {
		return Invalid("title must be at most %d characters", maxTitleLen)
	}
	return nil
}

func ValidateNewTask(title, description string, status Status) error {
	if err := validateTitle(title); err != nil {
		return err
	}
	if utf8.RuneCountInString(description) > maxDescriptionLen {
		return Invalid("description must be at most %d characters", maxDescriptionLen)
	}
	if status != "" && !status.Valid() {
		return Invalid("unknown status %q", status)
	}
	return nil
}

func AuthorizeTaskUpdate(actor Actor, task Task, changes map[string]FieldChange) error {
	if len(changes) == 0 {
		return nil
	}
	if actor.Role.CanEditAnyTask() {
		return nil
	}

	if task.CreatedBy == actor.UserID {
		return requireFields(changes, creatorFields,
			"task creators can change title, description, status and assignee only")
	}
	if task.AssigneeID != nil && *task.AssigneeID == actor.UserID {
		return requireFields(changes, assigneeFields,
			"assignees can change status only and cannot reassign task")
	}

	return Forbidden("only task creator, its assignee or a team admin can edit this task")
}

func requireFields(changes map[string]FieldChange, allowed []string, reason string) error {
	for field := range changes {
		if !slices.Contains(allowed, field) {
			return Forbidden("%s (rejected field %q)", reason, field)
		}
	}
	return nil
}