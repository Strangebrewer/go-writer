package project

import (
	"time"

	"github.com/Strangebrewer/go-writer/subject"
)

type Project struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Title       string     `json:"title"`
	SortOrder   int        `json:"sortOrder"`
	Description string     `json:"description"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

type ProjectResponse struct {
	Project
	Subjects []subject.Subject `json:"subjects"`
}

type CreateProjectRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
}

type UpdateProjectRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sortOrder"`
}
