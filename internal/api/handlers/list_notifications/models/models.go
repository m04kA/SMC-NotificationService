package models

import (
	"time"

	"github.com/m04kA/SMC-NotificationService/internal/domain"
	repoModels "github.com/m04kA/SMC-NotificationService/internal/infra/storage/notification"
	serviceModels "github.com/m04kA/SMC-NotificationService/internal/service/notifications/models"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

// ListNotificationsQuery параметры запроса для фильтрации списка уведомлений
type ListNotificationsQuery struct {
	Status         *string `json:"status,omitempty"`
	Type           *string `json:"type,omitempty"`
	TelegramUserID *int64  `json:"telegram_user_id,omitempty"`
	SpanID         *string `json:"span_id,omitempty"`
	Page           int     `json:"page"`
	Limit          int     `json:"limit"`
}

// Normalize устанавливает значения по умолчанию и валидирует параметры пагинации
func (q *ListNotificationsQuery) Normalize() {
	// Устанавливаем значения по умолчанию
	if q.Page <= 0 {
		q.Page = DefaultPage
	}

	if q.Limit <= 0 {
		q.Limit = DefaultLimit
	}

	// Ограничиваем максимальный размер страницы
	if q.Limit > MaxLimit {
		q.Limit = MaxLimit
	}
}

// ToRepositoryFilter преобразует HTTP query параметры в фильтр репозитория
func (q *ListNotificationsQuery) ToRepositoryFilter() *repoModels.ListFilter {
	filter := &repoModels.ListFilter{
		TelegramUserID: q.TelegramUserID,
		SpanID:         q.SpanID,
		Limit:          q.Limit,
		Offset:         (q.Page - 1) * q.Limit,
	}

	// Конвертируем строковые значения в ENUM типы
	if q.Status != nil {
		status := domain.NotificationStatus(*q.Status)
		filter.Status = &status
	}

	if q.Type != nil {
		notifType := domain.NotificationType(*q.Type)
		filter.Type = &notifType
	}

	return filter
}

// NotificationResponse HTTP ответ с данными уведомления
type NotificationResponse struct {
	ID             int64                     `json:"id"`
	TelegramUserID *int64                    `json:"telegram_user_id,omitempty"`
	ChatID         *int64                    `json:"chat_id,omitempty"`
	SpanID         *string                   `json:"span_id,omitempty"`
	MessageText    string                    `json:"message_text"`
	ImageURLs      []string                  `json:"image_urls,omitempty"`
	InlineButtons  []domain.InlineButton     `json:"inline_buttons,omitempty"`
	Type           domain.NotificationType   `json:"type"`
	Status         domain.NotificationStatus `json:"status"`
	ScheduledFor   *time.Time                `json:"scheduled_for,omitempty"`
	SentAt         *time.Time                `json:"sent_at,omitempty"`
	Metadata       domain.Metadata           `json:"metadata,omitempty"`
	ErrorMessage   *string                   `json:"error_message,omitempty"`
	RetryCount     int                       `json:"retry_count"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}

// FromDomainNotification преобразует доменную модель в HTTP ответ
func FromDomainNotification(n *domain.Notification) *NotificationResponse {
	return &NotificationResponse{
		ID:             n.ID,
		TelegramUserID: n.TelegramUserID,
		ChatID:         n.ChatID,
		SpanID:         n.SpanID,
		MessageText:    n.MessageText,
		ImageURLs:      n.ImageURLs,
		InlineButtons:  n.InlineButtons,
		Type:           n.Type,
		Status:         n.Status,
		ScheduledFor:   n.ScheduledFor,
		SentAt:         n.SentAt,
		Metadata:       n.Metadata,
		ErrorMessage:   n.ErrorMessage,
		RetryCount:     n.RetryCount,
		CreatedAt:      n.CreatedAt,
		UpdatedAt:      n.UpdatedAt,
	}
}

// ListNotificationsResponse HTTP ответ со списком уведомлений
type ListNotificationsResponse struct {
	Notifications []*NotificationResponse `json:"notifications"`
	Page          int                     `json:"page"`
	Limit         int                     `json:"limit"`
}

// FromServiceOutputs преобразует сервисные модели в HTTP ответ
func FromServiceOutputs(outputs []*serviceModels.NotificationOutput, page, limit int) *ListNotificationsResponse {
	notifications := make([]*NotificationResponse, len(outputs))
	for i, output := range outputs {
		notifications[i] = FromServiceOutput(output)
	}

	return &ListNotificationsResponse{
		Notifications: notifications,
		Page:          page,
		Limit:         limit,
	}
}

// FromServiceOutput преобразует сервисную модель в HTTP модель
func FromServiceOutput(output *serviceModels.NotificationOutput) *NotificationResponse {
	return &NotificationResponse{
		ID:             output.ID,
		TelegramUserID: output.TelegramUserID,
		ChatID:         output.ChatID,
		SpanID:         output.SpanID,
		MessageText:    output.MessageText,
		ImageURLs:      output.ImageURLs,
		InlineButtons:  output.InlineButtons,
		Type:           output.Type,
		Status:         output.Status,
		ScheduledFor:   output.ScheduledFor,
		SentAt:         output.SentAt,
		Metadata:       output.Metadata,
		ErrorMessage:   output.ErrorMessage,
		RetryCount:     output.RetryCount,
		CreatedAt:      output.CreatedAt,
		UpdatedAt:      output.UpdatedAt,
	}
}
