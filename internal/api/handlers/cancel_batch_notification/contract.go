package cancel_batch_notification

import (
	"context"
)

// NotificationService интерфейс сервиса уведомлений
type NotificationService interface {
	CancelBySpanID(ctx context.Context, spanID string) (int, error)
}

// Logger интерфейс для логирования
type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}
