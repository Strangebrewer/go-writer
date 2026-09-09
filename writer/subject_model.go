package writer

import (
	"time"
)

type Subject struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	SortOrder   int        `json:"sortOrder"`
	ProjectID   string     `json:"projectId"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

type SubjectResponse struct {
	Subject
	Texts []Text `json:"texts"`
}

type CreateSubjectRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
	ProjectID   string `json:"projectId"`
}

type UpdateSubjectRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sortOrder"`
	ProjectID   *string `json:"projectId"`
}
