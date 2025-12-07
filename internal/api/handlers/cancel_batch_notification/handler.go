package cancel_batch_notification

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers"
)

const (
	msgInvalidSpanID = "неверный span_id"
)

type Handler struct {
	service NotificationService
	logger  Logger
}

func NewHandler(service NotificationService, logger Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

type CancelBatchResponse struct {
	CancelledCount int `json:"cancelled_count"`
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Извлекаем span_id из URL параметров
	vars := mux.Vars(r)
	spanID := vars["span_id"]

	if spanID == "" {
		h.logger.Warn("Empty span_id provided")
		handlers.RespondBadRequest(w, msgInvalidSpanID)
		return
	}

	// Отменяем массовую рассылку через сервисный слой
	cancelledCount, err := h.service.CancelBySpanID(r.Context(), spanID)
	if err != nil {
		h.logger.Error("Failed to cancel batch notification %s: %v", spanID, err)
		handlers.RespondInternalError(w)
		return
	}

	h.logger.Info("Cancelled batch notification %s (%d notifications cancelled)", spanID, cancelledCount)

	// Возвращаем количество отмененных уведомлений
	handlers.RespondJSON(w, http.StatusOK, &CancelBatchResponse{
		CancelledCount: cancelledCount,
	})
}
