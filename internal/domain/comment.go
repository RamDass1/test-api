package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

const maxCommentLen = 5000

type Comment struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func ValidateComment(content string) error {
	if strings.TrimSpace(content) == "" {
		return Invalid("content must not be empty")
	}
	if utf8.RuneCountInString(content) > maxCommentLen {
		return Invalid("content must be at most %d characters", maxCommentLen)
	}
	return nil
}
