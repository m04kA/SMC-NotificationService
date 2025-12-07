package models

import (
	"time"

	"github.com/m04kA/SMC-NotificationService/internal/domain"
	serviceModels "github.com/m04kA/SMC-NotificationService/internal/service/notifications/models"
)

// CreateBatchNotificationRequest HTTP запрос на создание массовой рассылки
type CreateBatchNotificationRequest struct {
	TelegramUserIDs []int64                 `json:"telegram_user_ids"`
	MessageText     string                  `json:"message_text"`
	ImageURLs       []string                `json:"image_urls,omitempty"`
	InlineButtons   []domain.InlineButton   `json:"inline_buttons,omitempty"`
	Type            domain.NotificationType `json:"type"`
	ScheduledFor    *time.Time              `json:"scheduled_for,omitempty"`
	Metadata        domain.Metadata         `json:"metadata,omitempty"`
}

// ToServiceInput преобразует HTTP модель в сервисную модель
func (r *CreateBatchNotificationRequest) ToServiceInput() *serviceModels.CreateBatchNotificationInput {
	return &serviceModels.CreateBatchNotificationInput{
		TelegramUserIDs: r.TelegramUserIDs,
		MessageText:     r.MessageText,
		ImageURLs:       r.ImageURLs,
		InlineButtons:   r.InlineButtons,
		Type:            r.Type,
		ScheduledFor:    r.ScheduledFor,
		Metadata:        r.Metadata,
	}
}

// BatchNotificationResponse HTTP ответ на создание массовой рассылки
type BatchNotificationResponse struct {
	SpanID          string  `json:"span_id"`
	TotalCreated    int     `json:"total_created"`
	NotificationIDs []int64 `json:"notification_ids"`
	FailedUserIDs   []int64 `json:"failed_user_ids,omitempty"`
}

// FromServiceResult преобразует сервисный результат в HTTP ответ
func FromServiceResult(result *serviceModels.BatchNotificationResult) *BatchNotificationResponse {
	return &BatchNotificationResponse{
		SpanID:          result.SpanID,
		TotalCreated:    result.TotalCreated,
		NotificationIDs: result.NotificationIDs,
		FailedUserIDs:   result.FailedUserIDs,
	}
}
