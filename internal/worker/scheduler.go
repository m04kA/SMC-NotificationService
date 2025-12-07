package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/m04kA/SMC-NotificationService/internal/domain"
)

// Scheduler планировщик для отложенных уведомлений
type Scheduler struct {
	repo            NotificationRepository
	telegramService TelegramService
	logger          Logger
	scheduler       *gocron.Scheduler
	jobs            map[int64]*gocron.Job // notification_id -> job
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewScheduler создает новый экземпляр планировщика
func NewScheduler(repo NotificationRepository, telegramService TelegramService, logger Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		repo:            repo,
		telegramService: telegramService,
		logger:          logger,
		scheduler:       gocron.NewScheduler(time.UTC),
		jobs:            make(map[int64]*gocron.Job),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start запускает планировщик
func (s *Scheduler) Start() {
	s.logger.Info("Starting notification scheduler")
	s.scheduler.StartAsync()
}

// Stop останавливает планировщик
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping notification scheduler")
	s.cancel()
	s.scheduler.Stop()
	s.logger.Info("Notification scheduler stopped")
}

// LoadScheduledNotifications загружает все запланированные уведомления из БД при старте
// Критично для того, чтобы не терять отложенные сообщения при перезапуске сервиса
func (s *Scheduler) LoadScheduledNotifications(ctx context.Context) error {
	s.logger.Info("Loading scheduled notifications from database")

	notifications, err := s.repo.GetScheduledNotifications(ctx)
	if err != nil {
		return fmt.Errorf("failed to load scheduled notifications: %w", err)
	}

	s.logger.Info("Found %d scheduled notifications to load", len(notifications))

	for _, notification := range notifications {
		if err := s.ScheduleNotification(notification); err != nil {
			s.logger.Error("Failed to schedule notification %d: %v", notification.ID, err)
			continue
		}
	}

	s.logger.Info("Successfully loaded %d scheduled notifications", len(notifications))
	return nil
}

// ScheduleNotification планирует отправку уведомления на указанное время
func (s *Scheduler) ScheduleNotification(notification *domain.Notification) error {
	if notification.ScheduledFor == nil {
		return fmt.Errorf("notification %d has no scheduled_for time", notification.ID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, не запланировано ли уже это уведомление
	if _, exists := s.jobs[notification.ID]; exists {
		s.logger.Warn("Notification %d is already scheduled", notification.ID)
		return nil
	}

	// Создаем задачу на отправку
	job, err := s.scheduler.Every(1).StartAt(*notification.ScheduledFor).LimitRunsTo(1).Do(
		s.sendScheduledNotification,
		notification.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to schedule job for notification %d: %w", notification.ID, err)
	}

	s.jobs[notification.ID] = job
	s.logger.Info("Scheduled notification %d for %s", notification.ID, notification.ScheduledFor.Format(time.RFC3339))

	return nil
}

// CancelNotification отменяет запланированную отправку уведомления
func (s *Scheduler) CancelNotification(notificationID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[notificationID]
	if !exists {
		s.logger.Warn("Notification %d is not scheduled", notificationID)
		return nil
	}

	// Удаляем задачу из планировщика
	s.scheduler.RemoveByReference(job)
	delete(s.jobs, notificationID)

	s.logger.Info("Cancelled scheduled notification %d", notificationID)
	return nil
}

// sendScheduledNotification отправляет запланированное уведомление
// Вызывается планировщиком gocron в указанное время
func (s *Scheduler) sendScheduledNotification(notificationID int64) {
	s.logger.Info("Executing scheduled notification %d", notificationID)

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Получаем конкретное уведомление по ID
	notification, err := s.repo.GetByID(ctx, notificationID)
	if err != nil {
		s.logger.Error("Failed to fetch notification %d: %v", notificationID, err)
		s.removeJob(notificationID)
		return
	}

	// Проверяем, что уведомление всё ещё в статусе scheduled
	if notification.Status != domain.NotificationStatusScheduled {
		s.logger.Warn("Notification %d is no longer scheduled (current status: %s)", notificationID, notification.Status)
		s.removeJob(notificationID)
		return
	}

	// Обновляем статус на pending перед отправкой
	if err := s.repo.UpdateStatus(ctx, notificationID, domain.NotificationStatusPending); err != nil {
		s.logger.Error("Failed to update status for notification %d: %v", notificationID, err)
		return
	}

	// Преобразуем в Telegram сообщение
	// TelegramMessage содержит все нужные поля (ImageURLs, InlineButtons),
	// а TelegramService.SendMessage() автоматически определит тип отправки:
	// - текст (если нет изображений)
	// - фото (если 1 изображение)
	// - медиагруппа (если 2-10 изображений)
	telegramMsg := domain.NewTelegramMessage(notification)

	// Отправляем через Telegram API
	if err := s.telegramService.SendMessage(telegramMsg); err != nil {
		s.logger.Error("Failed to send notification %d: %v", notificationID, err)

		// Помечаем как failed с увеличением счётчика попыток
		if markErr := s.repo.MarkAsFailed(ctx, notificationID, err.Error(), true); markErr != nil {
			s.logger.Error("Failed to mark notification %d as failed: %v", notificationID, markErr)
		}

		s.removeJob(notificationID)
		return
	}

	// Помечаем как отправленное
	if err := s.repo.MarkAsSent(ctx, notificationID, time.Now()); err != nil {
		s.logger.Error("Failed to mark notification %d as sent: %v", notificationID, err)
	}

	s.logger.Info("Successfully sent scheduled notification %d", notificationID)
	s.removeJob(notificationID)
}

// removeJob удаляет задачу из внутреннего реестра (не из gocron)
func (s *Scheduler) removeJob(notificationID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, notificationID)
}
