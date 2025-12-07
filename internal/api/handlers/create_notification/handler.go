package create_notification

import (
	"errors"
	"net/http"

	"github.com/m04kA/SMC-NotificationService/internal/api/handlers"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers/create_notification/models"
	"github.com/m04kA/SMC-NotificationService/internal/domain"
	notificationsSvc "github.com/m04kA/SMC-NotificationService/internal/service/notifications"
)

const (
	msgInvalidRequestBody = "неверный формат тела запроса"
	msgInvalidRecipient   = "необходимо указать telegram_user_id или chat_id"
	msgUserNotFound       = "пользователь не найден в системе"
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
	// Парсинг request body
	var req models.CreateNotificationRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		h.logger.Warn("Failed to decode request body: %v", err)
		handlers.RespondBadRequest(w, msgInvalidRequestBody)
		return
	}

	// Валидация получателя
	if req.TelegramUserID == nil && req.ChatID == nil {
		h.logger.Warn("Missing recipient: neither telegram_user_id nor chat_id provided")
		handlers.RespondBadRequest(w, msgInvalidRecipient)
		return
	}

	// Создаём уведомление через сервисный слой
	notification, err := h.service.Create(r.Context(), req.ToServiceInput())
	if err != nil {
		// Обработка ошибок сервисного слоя
		if errors.Is(err, notificationsSvc.ErrInvalidRecipient) {
			handlers.RespondBadRequest(w, msgInvalidRecipient)
			return
		}
		if errors.Is(err, notificationsSvc.ErrUserNotFound) {
			handlers.RespondBadRequest(w, msgUserNotFound)
			return
		}

		h.logger.Error("Failed to create notification: %v", err)
		handlers.RespondInternalError(w)
		return
	}

	// Если уведомление отложенное - добавляем в scheduler
	if notification.Status == domain.NotificationStatusScheduled {
		if err := h.scheduler.ScheduleNotification(notification); err != nil {
			h.logger.Error("Failed to schedule notification %d: %v", notification.ID, err)
			// Не возвращаем ошибку клиенту, т.к. уведомление уже создано в БД
			// Worker сам подхватит его при следующей загрузке scheduled notifications
		}
	}

	h.logger.Info("Created notification with ID %d (type: %s, status: %s)", notification.ID, notification.Type, notification.Status)

	// Возвращаем созданное уведомление
	handlers.RespondJSON(w, http.StatusCreated, models.FromDomainNotification(notification))
}
