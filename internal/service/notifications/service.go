package notifications

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/m04kA/SMC-NotificationService/internal/domain"
	notificationRepo "github.com/m04kA/SMC-NotificationService/internal/infra/storage/notification"
	"github.com/m04kA/SMC-NotificationService/internal/integrations/userservice"
	"github.com/m04kA/SMC-NotificationService/internal/service/notifications/models"
)

// Service сервис для управления уведомлениями
type Service struct {
	notificationRepo NotificationRepository
	userServiceClient UserServiceClient
}

// NewService создает новый экземпляр сервиса уведомлений
func NewService(notificationRepo NotificationRepository, userServiceClient UserServiceClient) *Service {
	return &Service{
		notificationRepo:  notificationRepo,
		userServiceClient: userServiceClient,
	}
}

// Create создает одно уведомление
func (s *Service) Create(ctx context.Context, input *models.CreateNotificationInput) (*domain.Notification, error) {
	// Валидация получателя
	if input.TelegramUserID == nil && input.ChatID == nil {
		return nil, ErrInvalidRecipient
	}

	// Валидация пользователя в UserService (если указан telegram_user_id)
	if input.TelegramUserID != nil {
		if err := s.validateUser(ctx, *input.TelegramUserID); err != nil {
			return nil, fmt.Errorf("Create - %w", err)
		}
	}

	// Преобразуем в доменную модель
	notification := input.ToDomainNotification()

	// Создаем уведомление в БД
	id, err := s.notificationRepo.Create(ctx, notification)
	if err != nil {
		return nil, fmt.Errorf("%w: Create - repository error: %v", ErrInternal, err)
	}

	notification.ID = id
	return notification, nil
}

// CreateBatch создает массовую рассылку уведомлений
func (s *Service) CreateBatch(ctx context.Context, input *models.CreateBatchNotificationInput) (*models.BatchNotificationResult, error) {
	if len(input.TelegramUserIDs) == 0 {
		return nil, fmt.Errorf("%w: telegram_user_ids cannot be empty", ErrInvalidInput)
	}

	// Генерируем span_id для группировки массовой рассылки
	spanID := uuid.New().String()

	// Подготавливаем уведомления для всех получателей
	notifications := make([]*domain.Notification, 0, len(input.TelegramUserIDs))
	failedUserIDs := make([]int64, 0)

	for _, tgUserID := range input.TelegramUserIDs {
		// Валидация пользователя (пропускаем невалидных)
		if err := s.validateUser(ctx, tgUserID); err != nil {
			failedUserIDs = append(failedUserIDs, tgUserID)
			continue
		}

		notification := &domain.Notification{
			TelegramUserID: &tgUserID,
			SpanID:         &spanID,
			MessageText:    input.MessageText,
			ImageURLs:      input.ImageURLs,
			InlineButtons:  input.InlineButtons,
			Type:           input.Type,
			ScheduledFor:   input.ScheduledFor,
			Metadata:       input.Metadata,
		}

		// Определяем статус
		if input.ScheduledFor != nil {
			notification.Status = domain.NotificationStatusScheduled
		} else {
			notification.Status = domain.NotificationStatusPending
		}

		notifications = append(notifications, notification)
	}

	// Создаем все уведомления одним batch-запросом
	ids, err := s.notificationRepo.CreateBatch(ctx, notifications)
	if err != nil {
		return nil, fmt.Errorf("%w: CreateBatch - repository error: %v", ErrInternal, err)
	}

	return &models.BatchNotificationResult{
		SpanID:          spanID,
		TotalCreated:    len(ids),
		NotificationIDs: ids,
		FailedUserIDs:   failedUserIDs,
	}, nil
}

// GetByID получает уведомление по ID
func (s *Service) GetByID(ctx context.Context, id int64) (*domain.Notification, error) {
	notification, err := s.notificationRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, notificationRepo.ErrNotificationNotFound) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("%w: GetByID - repository error: %v", ErrInternal, err)
	}

	return notification, nil
}

// GetBySpanID получает все уведомления массовой рассылки
func (s *Service) GetBySpanID(ctx context.Context, spanID string) ([]*domain.Notification, error) {
	notifications, err := s.notificationRepo.GetBySpanID(ctx, spanID)
	if err != nil {
		return nil, fmt.Errorf("%w: GetBySpanID - repository error: %v", ErrInternal, err)
	}

	return notifications, nil
}

// List получает список уведомлений с фильтрацией
func (s *Service) List(ctx context.Context, filter models.ListNotificationsFilter) ([]*models.NotificationOutput, error) {
	repoFilter := notificationRepo.ListFilter{
		Status:         filter.Status,
		Type:           filter.Type,
		TelegramUserID: filter.TelegramUserID,
		SpanID:         filter.SpanID,
		Limit:          filter.Limit,
		Offset:         filter.Offset,
	}

	notifications, err := s.notificationRepo.List(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("%w: List - repository error: %v", ErrInternal, err)
	}

	// Конвертируем доменные модели в выходные модели сервиса
	outputs := make([]*models.NotificationOutput, len(notifications))
	for i, n := range notifications {
		outputs[i] = models.FromDomainNotification(n)
	}

	return outputs, nil
}

// Cancel отменяет одно уведомление
func (s *Service) Cancel(ctx context.Context, id int64) error {
	// Проверяем существование и возможность отмены
	notification, err := s.notificationRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, notificationRepo.ErrNotificationNotFound) {
			return ErrNotificationNotFound
		}
		return fmt.Errorf("%w: Cancel - repository error: %v", ErrInternal, err)
	}

	if !notification.CanBeCancelled() {
		return ErrCannotCancel
	}

	// Отменяем уведомление
	if err := s.notificationRepo.Cancel(ctx, id); err != nil {
		if errors.Is(err, notificationRepo.ErrNotificationNotFound) {
			return ErrNotificationNotFound
		}
		return fmt.Errorf("%w: Cancel - repository error: %v", ErrInternal, err)
	}

	return nil
}

// CancelBySpanID отменяет все уведомления массовой рассылки
func (s *Service) CancelBySpanID(ctx context.Context, spanID string) (int, error) {
	count, err := s.notificationRepo.CancelBySpanID(ctx, spanID)
	if err != nil {
		return 0, fmt.Errorf("%w: CancelBySpanID - repository error: %v", ErrInternal, err)
	}

	return count, nil
}

// validateUser проверяет существование пользователя в UserService
func (s *Service) validateUser(ctx context.Context, tgUserID int64) error {
	_, err := s.userServiceClient.GetUser(ctx, tgUserID)
	if err != nil {
		if errors.Is(err, userservice.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("%w: validateUser - userservice error: %v", ErrInternal, err)
	}

	return nil
}
