package domain

type AssigneeStat struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	ClosedTasks int64  `json:"closed_tasks"`
}

type TeamStats struct {
	TeamID          int64            `json:"team_id"`
	TasksByStatus   map[Status]int64 `json:"tasks_by_status"`
	TopAssignees30d []AssigneeStat   `json:"top_assignees_30d"`
	AvgCloseSeconds *float64         `json:"avg_close_seconds"`
	CommentsTotal   int64            `json:"comments_total"`
}
