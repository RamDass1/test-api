package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type Team struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	Role      Role      `json:"role,omitempty"`
}

type TeamMember struct {
	TeamID int64 `json:"team_id"`
	UserID int64 `json:"user_id"`
	Role   Role  `json:"role"`
	Email  string
	Name   string
}

type Actor struct {
	UserID int64
	Role   Role
}

func NormalizeTeamName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", Invalid("name must not be empty")
	}
	if utf8.RuneCountInString(name) > 255 {
		return "", Invalid("name is too long")
	}
	return name, nil
}
