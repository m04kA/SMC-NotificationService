package list_notifications

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/m04kA/SMC-NotificationService/internal/api/handlers"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers/list_notifications/models"
	serviceModels "github.com/m04kA/SMC-NotificationService/internal/service/notifications/models"
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

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Парсим query параметры
	query, err := h.parseQuery(r)
	if err != nil {
		h.logger.Warn("Invalid query parameters: %v", err)
		handlers.RespondBadRequest(w, err.Error())
		return
	}

	// Нормализуем параметры (устанавливаем значения по умолчанию)
	query.Normalize()

	// Преобразуем в фильтр сервисного слоя
	filter := query.ToRepositoryFilter()

	serviceFilter := serviceModels.ListNotificationsFilter{
		Status:         filter.Status,
		Type:           filter.Type,
		TelegramUserID: filter.TelegramUserID,
		SpanID:         filter.SpanID,
		Limit:          filter.Limit,
		Offset:         filter.Offset,
	}

	// Получаем список уведомлений
	outputs, err := h.service.List(r.Context(), serviceFilter)
	if err != nil {
		h.logger.Error("Failed to list notifications: %v", err)
		handlers.RespondInternalError(w)
		return
	}

	h.logger.Info("Listed %d notifications (page: %d, limit: %d)", len(outputs), query.Page, query.Limit)

	// Возвращаем результат
	handlers.RespondJSON(w, http.StatusOK, models.FromServiceOutputs(outputs, query.Page, query.Limit))
}

// parseQuery парсит query параметры из HTTP запроса
func (h *Handler) parseQuery(r *http.Request) (*models.ListNotificationsQuery, error) {
	queryParams := r.URL.Query()

	query := &models.ListNotificationsQuery{}

	// Парсим status
	if statusStr := queryParams.Get("status"); statusStr != "" {
		query.Status = &statusStr
	}

	// Парсим type
	if typeStr := queryParams.Get("type"); typeStr != "" {
		query.Type = &typeStr
	}

	// Парсим telegram_user_id
	if userIDStr := queryParams.Get("telegram_user_id"); userIDStr != "" {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid telegram_user_id: %s", userIDStr)
		}
		query.TelegramUserID = &userID
	}

	// Парсим span_id
	if spanID := queryParams.Get("span_id"); spanID != "" {
		query.SpanID = &spanID
	}

	// Парсим page
	if pageStr := queryParams.Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			return nil, fmt.Errorf("invalid page: %s", pageStr)
		}
		if page < 1 {
			return nil, fmt.Errorf("page must be >= 1")
		}
		query.Page = page
	}

	// Парсим limit
	if limitStr := queryParams.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %s", limitStr)
		}
		if limit < 1 {
			return nil, fmt.Errorf("limit must be >= 1")
		}
		query.Limit = limit
	}

	return query, nil
}
