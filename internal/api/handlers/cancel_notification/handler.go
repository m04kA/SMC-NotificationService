package cancel_notification

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers"
	notificationsSvc "github.com/m04kA/SMC-NotificationService/internal/service/notifications"
)

const (
	msgInvalidID            = "неверный ID уведомления"
	msgNotificationNotFound = "уведомление не найдено"
	msgCannotCancel         = "уведомление не может быть отменено (уже отправлено или завершилось с ошибкой)"
)

type Handler struct {
	service   NotificationService
	scheduler Scheduler
	logger    Logger
}

func NewHandler(service NotificationService, scheduler Scheduler, logger Logger) *Handler {
	return &Handler{
		service:   service,
		scheduler: scheduler,
		logger:    logger,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL параметров
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Warn("Invalid notification ID: %s", idStr)
		handlers.RespondBadRequest(w, msgInvalidID)
		return
	}

	// Отменяем уведомление через сервисный слой
	if err := h.service.Cancel(r.Context(), id); err != nil {
		// Обработка ошибок сервисного слоя
		if errors.Is(err, notificationsSvc.ErrNotificationNotFound) {
			handlers.RespondNotFound(w, msgNotificationNotFound)
			return
		}
		if errors.Is(err, notificationsSvc.ErrCannotCancel) {
			handlers.RespondBadRequest(w, msgCannotCancel)
			return
		}

		h.logger.Error("Failed to cancel notification %d: %v", id, err)
		handlers.RespondInternalError(w)
		return
	}

	// Отменяем задачу в scheduler (если она была запланирована)
	if err := h.scheduler.CancelNotification(id); err != nil {
		h.logger.Warn("Failed to cancel scheduled task for notification %d: %v", id, err)
		// Не возвращаем ошибку клиенту, т.к. статус в БД уже обновлен
	}

	h.logger.Info("Cancelled notification %d", id)

	// Возвращаем успех без тела ответа
	w.WriteHeader(http.StatusNoContent)
}
