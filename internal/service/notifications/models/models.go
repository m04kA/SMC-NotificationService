package models

import (
	"time"

	"github.com/m04kA/SMC-NotificationService/internal/domain"
)

// CreateNotificationInput входные данные для создания одного уведомления
type CreateNotificationInput struct {
	TelegramUserID *int64
	ChatID         *int64
	MessageText    string
	ImageURLs      []string
	InlineButtons  []domain.InlineButton
	Type           domain.NotificationType
	ScheduledFor   *time.Time
	Metadata       domain.Metadata
}

// CreateBatchNotificationInput входные данные для создания массовой рассылки
type CreateBatchNotificationInput struct {
	TelegramUserIDs []int64
	MessageText     string
	ImageURLs       []string
	InlineButtons   []domain.InlineButton
	Type            domain.NotificationType
	ScheduledFor    *time.Time
	Metadata        domain.Metadata
}

// BatchNotificationResult результат создания массовой рассылки
type BatchNotificationResult struct {
	SpanID          string
	TotalCreated    int
	NotificationIDs []int64
	FailedUserIDs   []int64
}

// ListNotificationsFilter фильтр для получения списка уведомлений
type ListNotificationsFilter struct {
	Status         *domain.NotificationStatus
	Type           *domain.NotificationType
	TelegramUserID *int64
	SpanID         *string
	Limit          int
	Offset         int
}

// NotificationOutput выходная модель для одного уведомления
type NotificationOutput struct {
	ID             int64
	TelegramUserID *int64
	ChatID         *int64
	SpanID         *string
	MessageText    string
	ImageURLs      []string // Преобразован из pq.StringArray
	InlineButtons  []domain.InlineButton
	Type           domain.NotificationType
	Status         domain.NotificationStatus
	ScheduledFor   *time.Time
	SentAt         *time.Time
	Metadata       domain.Metadata
	ErrorMessage   *string
	RetryCount     int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// FromDomainNotification преобразует доменную модель в выходную модель сервиса
func FromDomainNotification(n *domain.Notification) *NotificationOutput {
	return &NotificationOutput{
		ID:             n.ID,
		TelegramUserID: n.TelegramUserID,
		ChatID:         n.ChatID,
		SpanID:         n.SpanID,
		MessageText:    n.MessageText,
		ImageURLs:      []string(n.ImageURLs), // Приводим pq.StringArray к []string
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

// ToDomainNotification преобразует CreateNotificationInput в domain.Notification
func (input *CreateNotificationInput) ToDomainNotification() *domain.Notification {
	notification := &domain.Notification{
		TelegramUserID: input.TelegramUserID,
		ChatID:         input.ChatID,
		MessageText:    input.MessageText,
		ImageURLs:      input.ImageURLs,
		InlineButtons:  input.InlineButtons,
		Type:           input.Type,
		ScheduledFor:   input.ScheduledFor,
		Metadata:       input.Metadata,
	}

	// Определяем статус
	if input.ScheduledFor != nil && input.ScheduledFor.After(time.Now()) {
		notification.Status = domain.NotificationStatusScheduled
	} else {
		notification.Status = domain.NotificationStatusPending
	}

	return notification
}
