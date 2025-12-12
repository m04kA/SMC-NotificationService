package start_message

import (
	"context"

	"github.com/m04kA/SMC-NotificationService/internal/integrations/userservice"
)

// TelegramService интерфейс для работы с Telegram Bot API
type TelegramService interface {
	SendWelcomeMessage(chatID int64, tgUserID *int64) error
}

// UserServiceClient интерфейс для работы с UserService
type UserServiceClient interface {
	GetUser(ctx context.Context, tgUserID int64) (*userservice.User, error)
	CreateUser(ctx context.Context, req *userservice.CreateUserRequest) (*userservice.User, error)
}
