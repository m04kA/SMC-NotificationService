package create_notification

import (
	"context"

	"github.com/m04kA/SMC-NotificationService/internal/domain"
	"github.com/m04kA/SMC-NotificationService/internal/service/notifications/models"
)

// NotificationService интерфейс сервиса уведомлений
type NotificationService interface {
	Create(ctx context.Context, input *models.CreateNotificationInput) (*domain.Notification, error)
}

// Scheduler интерфейс планировщика для отложенных уведомлений
type Scheduler interface {
	ScheduleNotification(notification *domain.Notification) error
}

// Logger интерфейс для логирования
type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}
