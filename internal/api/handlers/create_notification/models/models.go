package models

import (
	"time"

	"github.com/m04kA/SMC-NotificationService/internal/domain"
	serviceModels "github.com/m04kA/SMC-NotificationService/internal/service/notifications/models"
)

// CreateNotificationRequest HTTP запрос на создание уведомления
type CreateNotificationRequest struct {
	TelegramUserID *int64                    `json:"telegram_user_id,omitempty"`
	ChatID         *int64                    `json:"chat_id,omitempty"`
	MessageText    string                    `json:"message_text"`
	ImageURLs      []string                  `json:"image_urls,omitempty"`
	InlineButtons  []domain.InlineButton     `json:"inline_buttons,omitempty"`
	Type           domain.NotificationType   `json:"type"`
	ScheduledFor   *time.Time                `json:"scheduled_for,omitempty"`
	Metadata       domain.Metadata           `json:"metadata,omitempty"`
}

// ToServiceInput преобразует HTTP модель в сервисную модель
func (r *CreateNotificationRequest) ToServiceInput() *serviceModels.CreateNotificationInput {
	return &serviceModels.CreateNotificationInput{
		TelegramUserID: r.TelegramUserID,
		ChatID:         r.ChatID,
		MessageText:    r.MessageText,
		ImageURLs:      r.ImageURLs,
		InlineButtons:  r.InlineButtons,
		Type:           r.Type,
		ScheduledFor:   r.ScheduledFor,
		Metadata:       r.Metadata,
	}
}

// NotificationResponse HTTP ответ с данными уведомления
type NotificationResponse struct {
	ID             int64                     `json:"id"`
	TelegramUserID *int64                    `json:"telegram_user_id,omitempty"`
	ChatID         *int64                    `json:"chat_id,omitempty"`
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
