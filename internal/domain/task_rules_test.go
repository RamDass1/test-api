package domain_test

import (
	"testing"

	"github.com/RamDass1/test-api/internal/domain"
)

func baseTask() domain.Task {
	id := int64(assignee)
	return domain.Task{
		ID:          10,
		TeamID:      1,
		Title:       "Ship report endpoint",
		Description: "one query, no N+1",
		Status:      domain.StatusTodo,
		CreatedBy:   creator,
		AssigneeID:  &id,
		Version:     3,
	}
}

func patchOf(fields map[string]any) domain.TaskPatch {
	var p domain.TaskPatch
	if v, ok := fields["title"]; ok {
		p.Title = domain.Optional[string]{Value: v.(string), Valid: true, Set: true}
	}
	if v, ok := fields["description"]; ok {
		p.Description = domain.Optional[string]{Value: v.(string), Valid: true, Set: true}
	}
	if v, ok := fields["status"]; ok {
		p.Status = domain.Optional[domain.Status]{Value: v.(domain.Status), Valid: true, Set: true}
	}
	if v, ok := fields["assignee_id"]; ok {
		p.AssigneeID = domain.Optional[int64]{Set: true}
		if v != nil {
			p.AssigneeID.Value = v.(int64)
			p.AssigneeID.Valid = true
		}
	}
	return p
}

func TestAuthorizeTaskUpdate(t *testing.T) {
	tests := []struct {
		name    string
		actor   domain.Actor
		patch   domain.TaskPatch
		wantErr domain.Code
	}{
		{
			name:  "owner edits any field of any task",
			actor: domain.Actor{UserID: owner, Role: domain.RoleOwner},
			patch: patchOf(map[string]any{"title": "new", "assignee_id": int64(owner)}),
		},
		{
			name:  "admin edits any field of any task",
			actor: domain.Actor{UserID: admin, Role: domain.RoleAdmin},
			patch: patchOf(map[string]any{"assignee_id": nil}),
		},
		{
			name:  "creator edits content and reassigns",
			actor: domain.Actor{UserID: creator, Role: domain.RoleMember},
			patch: patchOf(map[string]any{"title": "new", "description": "d", "assignee_id": int64(owner)}),
		},
		{
			name:  "assignee moves task forward",
			actor: domain.Actor{UserID: assignee, Role: domain.RoleMember},
			patch: patchOf(map[string]any{"status": domain.StatusInProgress}),
		},
		{
			name:    "assignee cannot reassign the task",
			actor:   domain.Actor{UserID: assignee, Role: domain.RoleMember},
			patch:   patchOf(map[string]any{"assignee_id": int64(stranger)}),
			wantErr: domain.CodeForbidden,
		},
		{
			name:    "assignee cannot rewrite title",
			actor:   domain.Actor{UserID: assignee, Role: domain.RoleMember},
			patch:   patchOf(map[string]any{"title": "new"}),
			wantErr: domain.CodeForbidden,
		},
		{
			name:    "unrelated member cannot edit",
			actor:   domain.Actor{UserID: stranger, Role: domain.RoleMember},
			patch:   patchOf(map[string]any{"status": domain.StatusDone}),
			wantErr: domain.CodeForbidden,
		},
		{
			name:  "unrelated member can submit patch that changes nothing",
			actor: domain.Actor{UserID: stranger, Role: domain.RoleMember},
			patch: patchOf(map[string]any{"status": domain.StatusTodo}),
		},
		{
			name:  "assignee repeating their own assignment is not a change",
			actor: domain.Actor{UserID: assignee, Role: domain.RoleMember},
			patch: patchOf(map[string]any{"assignee_id": int64(assignee)}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := baseTask()
			err := domain.AuthorizeTaskUpdate(tt.actor, task, domain.Diff(task, tt.patch))

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected update to be allowed, got %v", err)
				}
				return
			}
			assertCode(t, err, tt.wantErr)
		})
	}
}

func TestDiffIgnoresUnchangedValues(t *testing.T) {
	task := baseTask()
	patch := patchOf(map[string]any{
		"title":       task.Title,
		"description": task.Description,
		"status":      domain.StatusDone,
	})

	changes := domain.Diff(task, patch)

	if len(changes) != 1 {
		t.Fatalf("expected only status to change, got %v", changes)
	}
	if got := changes[domain.FieldStatus]; got.From != domain.StatusTodo || got.To != domain.StatusDone {
		t.Fatalf("unexpected status change recorded: %+v", got)
	}
}

func TestDiffDetectsClearedAssignee(t *testing.T) {
	task := baseTask()

	changes := domain.Diff(task, patchOf(map[string]any{"assignee_id": nil}))

	change, ok := changes[domain.FieldAssignee]
	if !ok {
		t.Fatal("clearing assignee should be recorded as a change")
	}
	if change.To != (*int64)(nil) {
		t.Fatalf("expected new assignee to be null, got %v", change.To)
	}
	if applied := domain.Apply(task, changes); applied.AssigneeID != nil {
		t.Fatalf("expected applied task to have no assignee, got %v", *applied.AssigneeID)
	}
}

func TestApplyMatchesTheRecordedChanges(t *testing.T) {
	task := baseTask()
	newAssignee := int64(owner)
	patch := patchOf(map[string]any{
		"title":       "  Ship it  ",
		"status":      domain.StatusInProgress,
		"assignee_id": newAssignee,
	})

	applied := domain.Apply(task, domain.Diff(task, patch))

	if applied.Title != "Ship it" {
		t.Errorf("title should be trimmed, got %q", applied.Title)
	}
	if applied.Status != domain.StatusInProgress {
		t.Errorf("unexpected status %q", applied.Status)
	}
	if applied.AssigneeID == nil || *applied.AssigneeID != newAssignee {
		t.Errorf("unexpected assignee %v", applied.AssigneeID)
	}
	if applied.Description != task.Description {
		t.Errorf("description should be untouched, got %q", applied.Description)
	}
}

func TestPatchValidation(t *testing.T) {
	tests := []struct {
		name  string
		patch domain.TaskPatch
		valid bool
	}{
		{name: "known status", patch: patchOf(map[string]any{"status": domain.StatusDone}), valid: true},
		{name: "unknown status", patch: patchOf(map[string]any{"status": domain.Status("archived")})},
		{name: "blank title", patch: patchOf(map[string]any{"title": "   "})},
		{name: "clearing assignee", patch: patchOf(map[string]any{"assignee_id": nil}), valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.patch.Validate()
			if tt.valid && err != nil {
				t.Fatalf("expected patch to be valid, got %v", err)
			}
			if !tt.valid {
				assertCode(t, err, domain.CodeValidation)
			}
		})
	}
}