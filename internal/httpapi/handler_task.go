package httpapi

import (
	"net/http"
	"strconv"

	"github.com/RamDass1/test-api/internal/domain"
	"github.com/RamDass1/test-api/internal/service"
)

type createTaskRequest struct {
	TeamID      int64  `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AssigneeID  *int64 `json:"assignee_id"`
}

type commentRequest struct {
	Content string `json:"content"`
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if req.TeamID <= 0 {
		writeError(w, r, domain.Invalid("team_id is required"))
		return
	}

	task, err := s.svc.CreateTask(r.Context(), userID(r.Context()), service.NewTaskRequest{
		TeamID:      req.TeamID,
		Title:       req.Title,
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	filter, err := taskFilterFrom(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	page, err := s.svc.ListTasks(r.Context(), userID(r.Context()), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func taskFilterFrom(r *http.Request) (domain.TaskFilter, error) {
	teamID, err := queryID(r, "team_id")
	if err != nil {
		return domain.TaskFilter{}, err
	}
	if teamID == nil {
		return domain.TaskFilter{}, domain.Invalid("team_id is required")
	}
	assigneeID, err := queryID(r, "assignee_id")
	if err != nil {
		return domain.TaskFilter{}, err
	}
	limit, err := queryInt(r, "limit", service.DefaultLimit)
	if err != nil {
		return domain.TaskFilter{}, err
	}
	offset, err := queryInt(r, "offset", 0)
	if err != nil {
		return domain.TaskFilter{}, err
	}

	filter := domain.TaskFilter{
		TeamID:     *teamID,
		AssigneeID: assigneeID,
		Limit:      limit,
		Offset:     offset,
	}
	if raw := r.URL.Query().Get("status"); raw != "" {
		status := domain.Status(raw)
		filter.Status = &status
	}
	return filter, nil
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathID(r, "task_id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	var patch domain.TaskPatch
	if err := decodeJSON(w, r, &patch); err != nil {
		writeError(w, r, err)
		return
	}
	if patch.Empty() && patch.Version == nil {
		writeError(w, r, domain.Invalid("provide at least one field to update"))
		return
	}

	task, err := s.svc.UpdateTask(r.Context(), userID(r.Context()), taskID, patch)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleTaskHistory(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathID(r, "task_id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	entries, err := s.svc.TaskHistory(r.Context(), userID(r.Context()), taskID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": entries})
}

func (s *Server) handleAddComment(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathID(r, "task_id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	var req commentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	comment, err := s.svc.AddComment(r.Context(), userID(r.Context()), taskID, req.Content)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

func (s *Server) handleListComments(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathID(r, "task_id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	comments, err := s.svc.ListComments(r.Context(), userID(r.Context()), taskID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": comments})
}

func queryID(r *http.Request, name string) (*int64, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil, domain.Invalid("%s must be a positive integer", name)
	}
	return &id, nil
}

func queryInt(r *http.Request, name string, fallback int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, domain.Invalid("%s must be a non-negative integer", name)
	}
	return value, nil
}
