package health

import (
	"net/http"

	"github.com/m04kA/SMC-NotificationService/internal/api/handlers"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "healthy",
	}

	handlers.RespondJSON(w, http.StatusOK, response)
}
