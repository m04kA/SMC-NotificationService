package notification

import "github.com/m04kA/SMC-NotificationService/internal/domain"

// ListFilter параметры для фильтрации списка уведомлений
type ListFilter struct {
	Status         *domain.NotificationStatus
	Type           *domain.NotificationType
	TelegramUserID *int64
	SpanID         *string
	Limit          int
	Offset         int
}
