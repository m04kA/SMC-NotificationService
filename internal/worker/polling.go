package worker

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// PollingHandler обрабатывает входящие сообщения от Telegram в режиме long polling
type PollingHandler struct {
	startMessageUseCase StartMessageUseCase
	logger              Logger
}

// NewPollingHandler создаёт новый обработчик для long polling
func NewPollingHandler(startMessageUseCase StartMessageUseCase, logger Logger) *PollingHandler {
	return &PollingHandler{
		startMessageUseCase: startMessageUseCase,
		logger:              logger,
	}
}

// Start запускает обработку обновлений из канала
// Блокирующий метод, должен вызываться в отдельной goroutine
func (h *PollingHandler) Start(ctx context.Context, updatesChan tgbotapi.UpdatesChannel) {
	h.logger.Info("Starting Telegram long polling handler...")

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Stopping Telegram long polling handler...")
			return

		case update := <-updatesChan:
			h.handleUpdate(ctx, update)
		}
	}
}

// handleUpdate обрабатывает одно обновление от Telegram
func (h *PollingHandler) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	// Обрабатываем только текстовые сообщения
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	// Обрабатываем только команду /start
	if update.Message.Text != "/start" {
		return
	}

	chatID := update.Message.Chat.ID
	tgUserID := update.Message.From.ID

	h.logger.Info("Received /start command from user %d (chat %d)", tgUserID, chatID)

	// Обрабатываем команду через use case
	if err := h.startMessageUseCase.Execute(ctx, update.Message.From, chatID); err != nil {
		h.logger.Error("Failed to handle /start command for user %d: %v", tgUserID, err)
		return
	}

	h.logger.Info("Successfully processed /start command for user %d (chat %d)", tgUserID, chatID)
}
