package notifications

import (
	"context"

	"github.com/m04kA/SMC-NotificationService/internal/domain"
	notificationRepo "github.com/m04kA/SMC-NotificationService/internal/infra/storage/notification"
	"github.com/m04kA/SMC-NotificationService/internal/integrations/userservice"
)

// NotificationRepository интерфейс репозитория уведомлений
type NotificationRepository interface {
	Create(ctx context.Context, notification *domain.Notification) (int64, error)
	CreateBatch(ctx context.Context, notifications []*domain.Notification) ([]int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Notification, error)
	GetBySpanID(ctx context.Context, spanID string) ([]*domain.Notification, error)
	List(ctx context.Context, filter notificationRepo.ListFilter) ([]*domain.Notification, error)
	Cancel(ctx context.Context, id int64) error
	CancelBySpanID(ctx context.Context, spanID string) (int, error)
}

// UserServiceClient интерфейс клиента UserService
type UserServiceClient interface {
	GetUser(ctx context.Context, tgUserID int64) (*userservice.User, error)
}
