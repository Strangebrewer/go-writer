package server

import (
	"net/http"

	"github.com/Strangebrewer/go-writer/app"
	"github.com/Strangebrewer/go-writer/health"
	"github.com/Strangebrewer/go-writer/writer"
	"github.com/go-chi/chi/v5"
)

func registerRoutes(r chi.Router, application *app.Application, authMiddleware func(http.Handler) http.Handler) {
	r.Get("/health", health.Handler)

	r.With(authMiddleware).Mount("/projects",
		writer.ProjectRoutes(application.ProjectStore, application.SubjectStore, application.Tracer),
	)
	r.With(authMiddleware).Mount("/subjects",
		writer.SubjectRoutes(application.SubjectStore, application.TextStore, application.Tracer),
	)
	r.With(authMiddleware).Mount("/texts", writer.TextRoutes(application.TextStore, application.Tracer))
}
