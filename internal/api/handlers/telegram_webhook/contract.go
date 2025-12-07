package telegram_webhook

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
