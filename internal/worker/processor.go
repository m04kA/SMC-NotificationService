package worker

import (
	"context"
	"sync"
	"time"

	"github.com/m04kA/SMC-NotificationService/internal/domain"
)

// Processor обработчик pending уведомлений
type Processor struct {
	repo            NotificationRepository
	telegramService TelegramService
	logger          Logger
	interval        time.Duration // Интервал опроса БД (по умолчанию 30 секунд)
	batchSize       int           // Количество уведомлений за один опрос
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// NewProcessor создает новый экземпляр обработчика
func NewProcessor(repo NotificationRepository, telegramService TelegramService, logger Logger, interval time.Duration, batchSize int) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Processor{
		repo:            repo,
		telegramService: telegramService,
		logger:          logger,
		interval:        interval,
		batchSize:       batchSize,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start запускает обработчик в отдельной goroutine
func (p *Processor) Start() {
	p.logger.Info("Starting notification processor (interval: %s, batch size: %d)", p.interval, p.batchSize)

	p.wg.Add(1)
	go p.run()
}

// Stop останавливает обработчик
func (p *Processor) Stop() {
	p.logger.Info("Stopping notification processor")
	p.cancel()
	p.wg.Wait()
	p.logger.Info("Notification processor stopped")
}

// SetInterval устанавливает интервал опроса БД
func (p *Processor) SetInterval(interval time.Duration) {
	p.interval = interval
}

// SetBatchSize устанавливает размер батча для обработки
func (p *Processor) SetBatchSize(size int) {
	p.batchSize = size
}

// run основной цикл обработки pending уведомлений
func (p *Processor) run() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Первый запуск сразу (не ждём первого тика)
	p.processPendingNotifications()

	for {
		select {
		case <-ticker.C:
			p.processPendingNotifications()
		case <-p.ctx.Done():
			return
		}
	}
}

// processPendingNotifications обрабатывает очередь pending уведомлений
func (p *Processor) processPendingNotifications() {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Minute)
	defer cancel()

	// Получаем pending уведомления из БД
	notifications, err := p.repo.GetPendingNotifications(ctx, p.batchSize)
	if err != nil {
		p.logger.Error("Failed to fetch pending notifications: %v", err)
		return
	}

	if len(notifications) == 0 {
		return
	}

	p.logger.Info("Processing %d pending notifications", len(notifications))

	// Обрабатываем каждое уведомление
	for _, notification := range notifications {
		// Проверяем, не был ли отменён контекст
		select {
		case <-p.ctx.Done():
			p.logger.Info("Processor stopped, aborting notification processing")
			return
		default:
		}

		p.processNotification(ctx, notification)
	}

	p.logger.Info("Finished processing batch of %d notifications", len(notifications))
}

// processNotification обрабатывает одно уведомление
func (p *Processor) processNotification(ctx context.Context, notification *domain.Notification) {
	p.logger.Info("Processing notification %d (type: %s, user: %v, chat: %v)",
		notification.ID,
		notification.Type,
		notification.TelegramUserID,
		notification.ChatID,
	)

	// Преобразуем в Telegram сообщение
	telegramMsg := domain.NewTelegramMessage(notification)

	// Отправляем через Telegram API
	if err := p.telegramService.SendMessage(telegramMsg); err != nil {
		p.logger.Error("Failed to send notification %d: %v", notification.ID, err)

		// Помечаем как failed с увеличением счётчика попыток
		if markErr := p.repo.MarkAsFailed(ctx, notification.ID, err.Error(), true); markErr != nil {
			p.logger.Error("Failed to mark notification %d as failed: %v", notification.ID, markErr)
		}

		return
	}

	// Помечаем как отправленное
	if err := p.repo.MarkAsSent(ctx, notification.ID, time.Now()); err != nil {
		p.logger.Error("Failed to mark notification %d as sent: %v", notification.ID, err)
		return
	}

	p.logger.Info("Successfully sent notification %d", notification.ID)
}
