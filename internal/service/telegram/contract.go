package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// BotAPI интерфейс для Telegram Bot API
// Абстракция над tgbotapi.BotAPI для упрощения тестирования
type BotAPI interface {
	// Send отправляет сообщение через Telegram Bot API
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)

	// Request выполняет кастомный запрос к Telegram API
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)

	// GetUpdatesChan возвращает канал для получения обновлений (long polling)
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
}
