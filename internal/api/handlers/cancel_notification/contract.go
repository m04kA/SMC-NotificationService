package cancel_notification

import (
	"context"
)

// NotificationService интерфейс сервиса уведомлений
type NotificationService interface {
	Cancel(ctx context.Context, id int64) error
}

// Scheduler интерфейс планировщика для отмены отложенных уведомлений
type Scheduler interface {
	CancelNotification(notificationID int64) error
}

// Logger интерфейс для логирования
type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}
