package start_message

import (
	"context"
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/m04kA/SMC-NotificationService/internal/integrations/userservice"
)

const (
	// DefaultUserRole роль по умолчанию для новых пользователей
	DefaultUserRole = "client"
)

// UseCase обрабатывает команду /start
type UseCase struct {
	telegramService   TelegramService
	userServiceClient UserServiceClient
}

// New создаёт новый use case для обработки /start
func New(telegramService TelegramService, userServiceClient UserServiceClient) *UseCase {
	return &UseCase{
		telegramService:   telegramService,
		userServiceClient: userServiceClient,
	}
}

// Execute выполняет обработку команды /start
// Возвращает ошибку с полным контекстом для логирования на уровне выше
func (uc *UseCase) Execute(ctx context.Context, from *tgbotapi.User, chatID int64) error {
	// Определяем tgUserID (может быть nil если from == nil)
	var tgUserID *int64
	if from != nil {
		userID := from.ID
		tgUserID = &userID
	}

	// Отправляем приветственное сообщение сразу с tgUserID
	if err := uc.telegramService.SendWelcomeMessage(chatID, tgUserID); err != nil {
		return fmt.Errorf("usecase.SendStartMessage: send welcome message to chat %d: %w", chatID, err)
	}

	// Проверяем существование пользователя и создаём при необходимости
	if from != nil {
		if err := uc.ensureUserExists(ctx, from); err != nil {
			return fmt.Errorf("usecase.SendStartMessage: ensure user %d exists: %w", from.ID, err)
		}
	}

	return nil
}

// ensureUserExists проверяет существование пользователя и создаёт его при необходимости
func (uc *UseCase) ensureUserExists(ctx context.Context, from *tgbotapi.User) error {
	tgUserID := from.ID

	// Проверяем, существует ли пользователь
	_, err := uc.userServiceClient.GetUser(ctx, tgUserID)
	if err == nil {
		return nil // Пользователь уже существует
	}

	// Если пользователь не найден - создаём
	if !errors.Is(err, userservice.ErrUserNotFound) {
		return fmt.Errorf("check user existence: %w", err)
	}

	// Формируем имя пользователя
	userName := from.FirstName
	if from.LastName != "" {
		userName += " " + from.LastName
	}

	// Формируем tg_link если есть username
	var tgLink *string
	if from.UserName != "" {
		link := "@" + from.UserName
		tgLink = &link
	}

	// Создаём пользователя
	createReq := &userservice.CreateUserRequest{
		TgUserID: tgUserID,
		Name:     userName,
		TgLink:   tgLink,
		Role:     DefaultUserRole,
	}

	if _, err := uc.userServiceClient.CreateUser(ctx, createReq); err != nil {
		return fmt.Errorf("create user '%s': %w", userName, err)
	}

	return nil
}
