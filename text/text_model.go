package text

import (
	"time"

	"github.com/google/uuid"
)

type Text struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Content     string     `json:"content"`
	SubjectID   string     `json:"subjectId"`
	ProjectID   string     `json:"projectId"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

type CreateTextRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	SubjectID   string `json:"subjectId"`
	ProjectID   string `json:"projectId"`
}

type UpdateTextRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Content     *string `json:"content"`
	SubjectID   *string `json:"subjectId"`
	ProjectID   *string `json:"projectId"`
}

func newID() (uuid.UUID, error) {
	return uuid.NewV7()
}
