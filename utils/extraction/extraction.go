package extraction

import (
	"errors"
	"net/http"

	"github.com/Strangebrewer/go-writer/middleware"
	"github.com/google/uuid"
)

func UserIDFromRequest(r *http.Request) (uuid.UUID, error) {
	idStr, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return uuid.UUID{}, errors.New("no user id in context")
	}
	return uuid.Parse(idStr)
}
