package worker

import (
	"context"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/m04kA/SMC-NotificationService/internal/domain"
)

// NotificationRepository интерфейс для работы с репозиторием уведомлений
type NotificationRepository interface {
	// GetScheduledNotifications получает все запланированные уведомления для загрузки в scheduler
	GetScheduledNotifications(ctx context.Context) ([]*domain.Notification, error)

	// GetPendingNotifications получает все уведомления со статусом pending для немедленной отправки
	GetPendingNotifications(ctx context.Context, limit int) ([]*domain.Notification, error)

	// UpdateStatus обновляет статус уведомления
	UpdateStatus(ctx context.Context, id int64, status domain.NotificationStatus) error

	// MarkAsSent помечает уведомление как отправленное
	MarkAsSent(ctx context.Context, id int64, sentAt time.Time) error

	// MarkAsFailed помечает уведомление как неудачное
	// Параметр incrementRetry указывает, нужно ли увеличить счётчик попыток
	MarkAsFailed(ctx context.Context, id int64, errorMsg string, incrementRetry bool) error

	// GetByID получает уведомление по ID
	GetByID(ctx context.Context, id int64) (*domain.Notification, error)
}

// TelegramService интерфейс для отправки сообщений через Telegram Bot API
type TelegramService interface {
	// SendMessage отправляет уведомление через Telegram
	SendMessage(msg *domain.TelegramMessage) error
}

// StartMessageUseCase интерфейс для обработки команды /start
type StartMessageUseCase interface {
	Execute(ctx context.Context, from *tgbotapi.User, chatID int64) error
}

// Logger интерфейс для логирования
type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}
