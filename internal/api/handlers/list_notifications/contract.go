package list_notifications

import (
	"context"

	serviceModels "github.com/m04kA/SMC-NotificationService/internal/service/notifications/models"
)

// NotificationService интерфейс сервиса уведомлений
type NotificationService interface {
	List(ctx context.Context, filter serviceModels.ListNotificationsFilter) ([]*serviceModels.NotificationOutput, error)
}

// Logger интерфейс для логирования
type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}
