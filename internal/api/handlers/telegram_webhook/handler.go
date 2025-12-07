package telegram_webhook

import (
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers"
)

const (
	msgInvalidRequestBody = "неверный формат тела запроса"
	commandStart          = "/start"
)

type Handler struct {
	startMessageUseCase StartMessageUseCase
	logger              Logger
}

func NewHandler(startMessageUseCase StartMessageUseCase, logger Logger) *Handler {
	return &Handler{
		startMessageUseCase: startMessageUseCase,
		logger:              logger,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Парсим webhook update от Telegram
	var update tgbotapi.Update
	if err := handlers.DecodeJSON(r, &update); err != nil {
		h.logger.Warn("Failed to decode telegram webhook: %v", err)
		handlers.RespondBadRequest(w, msgInvalidRequestBody)
		return
	}

	// Обрабатываем только текстовые сообщения
	if update.Message == nil || update.Message.Text == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Обрабатываем только команду /start
	if update.Message.Text != commandStart {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Обработка команды /start через use case
	chatID := update.Message.Chat.ID
	tgUserID := update.Message.From.ID

	if err := h.startMessageUseCase.Execute(r.Context(), update.Message.From, chatID); err != nil {
		h.logger.Error("Failed to handle /start command for user %d: %v", tgUserID, err)
		handlers.RespondInternalError(w)
		return
	}

	h.logger.Info("Successfully processed /start command for user %d (chat %d)", tgUserID, chatID)
	w.WriteHeader(http.StatusOK)
}
