package create_batch_notification

import (
	"errors"
	"net/http"

	"github.com/m04kA/SMC-NotificationService/internal/api/handlers"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers/create_batch_notification/models"
	"github.com/m04kA/SMC-NotificationService/internal/domain"
	notificationsSvc "github.com/m04kA/SMC-NotificationService/internal/service/notifications"
)

const (
	msgInvalidRequestBody = "неверный формат тела запроса"
	msgEmptyRecipientList = "список telegram_user_ids не может быть пустым"
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
	var req models.CreateBatchNotificationRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		h.logger.Warn("Failed to decode request body: %v", err)
		handlers.RespondBadRequest(w, msgInvalidRequestBody)
		return
	}

	// Валидация списка получателей
	if len(req.TelegramUserIDs) == 0 {
		h.logger.Warn("Empty telegram_user_ids list")
		handlers.RespondBadRequest(w, msgEmptyRecipientList)
		return
	}

	// Создаём массовую рассылку через сервисный слой
	result, err := h.service.CreateBatch(r.Context(), req.ToServiceInput())
	if err != nil {
		// Обработка ошибок сервисного слоя
		if errors.Is(err, notificationsSvc.ErrInvalidInput) {
			handlers.RespondBadRequest(w, err.Error())
			return
		}

		h.logger.Error("Failed to create batch notification: %v", err)
		handlers.RespondInternalError(w)
		return
	}

	h.logger.Info("Created batch notification with span_id=%s (created: %d, failed: %d)",
		result.SpanID, result.TotalCreated, len(result.FailedUserIDs))

	// Если уведомления отложенные - добавляем их в scheduler
	if req.ScheduledFor != nil {
		// Загружаем созданные уведомления по span_id
		notifications, err := h.service.GetBySpanID(r.Context(), result.SpanID)
		if err != nil {
			h.logger.Error("Failed to load notifications for scheduling (span_id=%s): %v", result.SpanID, err)
			// Не возвращаем ошибку, т.к. уведомления уже созданы в БД
		} else {
			scheduledCount := 0
			for _, notification := range notifications {
				if notification.Status == domain.NotificationStatusScheduled {
					if err := h.scheduler.ScheduleNotification(notification); err != nil {
						h.logger.Error("Failed to schedule notification %d: %v", notification.ID, err)
						// Продолжаем планировать остальные
					} else {
						scheduledCount++
					}
				}
			}
			h.logger.Info("Scheduled %d/%d notifications from batch %s", scheduledCount, len(notifications), result.SpanID)
		}
	}

	// Возвращаем результат массовой рассылки
	handlers.RespondJSON(w, http.StatusCreated, models.FromServiceResult(result))
}
