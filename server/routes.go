package server

import (
	"net/http"

	"github.com/Strangebrewer/go-writer/app"
	"github.com/Strangebrewer/go-writer/health"
	"github.com/Strangebrewer/go-writer/project"
	"github.com/Strangebrewer/go-writer/subject"
	"github.com/Strangebrewer/go-writer/text"
	"github.com/go-chi/chi/v5"
)

func registerRoutes(r chi.Router, application *app.Application, authMiddleware func(http.Handler) http.Handler) {
	r.Get("/health", health.Handler)

	r.With(authMiddleware).Mount("/projects",
		project.Routes(application.ProjectStore, application.SubjectStore, application.Tracer),
	)
	r.With(authMiddleware).Mount("/subjects",
		subject.Routes(application.SubjectStore, application.TextStore, application.Tracer),
	)
	r.With(authMiddleware).Mount("/texts", text.Routes(application.TextStore, application.Tracer))
}
