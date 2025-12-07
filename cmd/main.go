package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/m04kA/SMC-NotificationService/internal/api/handlers/cancel_batch_notification"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers/cancel_notification"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers/create_batch_notification"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers/create_notification"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers/health"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers/list_notifications"
	"github.com/m04kA/SMC-NotificationService/internal/api/handlers/telegram_webhook"
	"github.com/m04kA/SMC-NotificationService/internal/api/middleware"
	"github.com/m04kA/SMC-NotificationService/internal/config"
	"github.com/m04kA/SMC-NotificationService/internal/infra/storage/notification"
	"github.com/m04kA/SMC-NotificationService/internal/integrations/userservice"
	"github.com/m04kA/SMC-NotificationService/internal/service/notifications"
	"github.com/m04kA/SMC-NotificationService/internal/service/telegram"
	"github.com/m04kA/SMC-NotificationService/internal/usecase/start_message"
	"github.com/m04kA/SMC-NotificationService/internal/worker"
	"github.com/m04kA/SMC-NotificationService/pkg/dbmetrics"
	"github.com/m04kA/SMC-NotificationService/pkg/logger"
	"github.com/m04kA/SMC-NotificationService/pkg/metrics"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load("config.toml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Инициализируем логгер
	log, err := logger.New(cfg.Logs.File, cfg.Logs.Level)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	log.Info("Starting SMC-NotificationService...")
	log.Info("Configuration loaded from config.toml")

	// Инициализируем метрики (если включены)
	var metricsCollector *metrics.Metrics
	var wrappedDB *dbmetrics.DB
	stopMetricsCh := make(chan struct{})

	if cfg.Metrics.Enabled {
		metricsCollector = metrics.New(cfg.Metrics.ServiceName)
		log.Info("Metrics enabled at %s", cfg.Metrics.Path)
	}

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		log.Fatal("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Настраиваем connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database: %v", err)
	}
	log.Info("Successfully connected to database (host=%s, port=%d, db=%s)",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// Инициализируем repository
	var notificationRepo *notification.Repository

	if cfg.Metrics.Enabled {
		wrappedDB = dbmetrics.WrapWithDefault(db, metricsCollector, cfg.Metrics.ServiceName, stopMetricsCh)
		log.Info("Database metrics collection started")
		notificationRepo = notification.NewRepository(wrappedDB)
	} else {
		notificationRepo = notification.NewRepository(db)
	}

	// Создаём контекст с возможностью отмены для управления жизненным циклом горутин
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	// Инициализируем интеграцию с UserService
	userServiceClient := userservice.NewClient(
		cfg.UserService.URL,
		time.Duration(cfg.UserService.Timeout)*time.Second,
	)
	log.Info("UserService client initialized (url=%s)", cfg.UserService.URL)

	// Инициализируем Telegram Bot API
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		log.Fatal("Failed to initialize Telegram Bot API: %v", err)
	}
	log.Info("Telegram Bot API initialized (@%s)", bot.Self.UserName)

	// Инициализируем Telegram Service
	telegramSvc := telegram.NewService(bot)
	log.Info("Telegram service initialized")

	// Инициализируем use case для обработки /start
	startMessageUC := start_message.New(telegramSvc, userServiceClient)
	log.Info("Start message use case initialized")

	// Определяем режим работы: Webhook или Long Polling
	if cfg.Telegram.WebhookURL != "" {
		// Режим Webhook
		log.Info("Using Webhook mode")

		if err := telegramSvc.SetWebhook(cfg.Telegram.WebhookURL); err != nil {
			log.Fatal("Failed to set Telegram webhook: %v", err)
		}
		log.Info("Telegram webhook set to %s", cfg.Telegram.WebhookURL)
	} else {
		// Режим Long Polling
		log.Info("Using Long Polling mode")

		if err := telegramSvc.DeleteWebhook(); err != nil {
			log.Warn("Failed to delete webhook (may not exist): %v", err)
		}

		// Создаём polling handler
		pollingHandler := worker.NewPollingHandler(startMessageUC, log)

		// Запускаем long polling в фоне
		updatesChan := telegramSvc.GetUpdatesChan(0)
		go pollingHandler.Start(ctx, updatesChan)
		log.Info("Telegram long polling started")
	}

	// Инициализируем Notifications Service
	notificationSvc := notifications.NewService(notificationRepo, userServiceClient)
	log.Info("Notification service initialized")

	// Инициализируем Worker компоненты
	scheduler := worker.NewScheduler(notificationRepo, telegramSvc, log)
	processor := worker.NewProcessor(
		notificationRepo,
		telegramSvc,
		log,
		time.Duration(cfg.Worker.ProcessorInterval)*time.Second,
		cfg.Worker.ProcessorBatchSize,
	)

	// КРИТИЧНО: Запускаем scheduler ПЕРЕД загрузкой notifications
	scheduler.Start()
	log.Info("Notification scheduler started")

	// КРИТИЧНО: Загружаем отложенные уведомления из БД при старте
	if err := scheduler.LoadScheduledNotifications(ctx); err != nil {
		log.Error("Failed to load scheduled notifications: %v", err)
	} else {
		log.Info("Scheduled notifications loaded successfully")
	}

	// Запускаем processor в фоне
	go processor.Start()
	log.Info("Notification processor started (interval=%ds, batch=%d)",
		cfg.Worker.ProcessorInterval, cfg.Worker.ProcessorBatchSize)

	// Инициализируем handlers
	healthHandler := health.NewHandler()
	createNotificationHandler := create_notification.NewHandler(notificationSvc, scheduler, log)
	createBatchNotificationHandler := create_batch_notification.NewHandler(notificationSvc, scheduler, log)
	listNotificationsHandler := list_notifications.NewHandler(notificationSvc, log)
	cancelNotificationHandler := cancel_notification.NewHandler(notificationSvc, scheduler, log)
	cancelBatchNotificationHandler := cancel_batch_notification.NewHandler(notificationSvc, log)
	telegramWebhookHandler := telegram_webhook.NewHandler(startMessageUC, log)

	// Настраиваем роутер
	r := mux.NewRouter()

	// Добавляем metrics middleware (если метрики включены)
	if cfg.Metrics.Enabled {
		r.Use(middleware.MetricsMiddleware(metricsCollector, cfg.Metrics.ServiceName))
		log.Info("HTTP metrics middleware enabled")
	}

	// Публичные endpoints
	r.HandleFunc("/health", healthHandler.Handle).Methods(http.MethodGet)
	r.HandleFunc("/webhook/telegram", telegramWebhookHandler.Handle).Methods(http.MethodPost)

	// Metrics endpoint (публичный)
	if cfg.Metrics.Enabled {
		r.Handle(cfg.Metrics.Path, promhttp.Handler()).Methods(http.MethodGet)
		log.Info("Prometheus metrics endpoint exposed at %s", cfg.Metrics.Path)
	}

	// API v1 endpoints
	api := r.PathPrefix("/api/v1").Subrouter()

	// Notifications endpoints
	api.HandleFunc("/notifications", createNotificationHandler.Handle).Methods(http.MethodPost)
	api.HandleFunc("/notifications/batch", createBatchNotificationHandler.Handle).Methods(http.MethodPost)
	api.HandleFunc("/notifications", listNotificationsHandler.Handle).Methods(http.MethodGet)
	api.HandleFunc("/notifications/{id}", cancelNotificationHandler.Handle).Methods(http.MethodDelete)
	api.HandleFunc("/notifications/batch/{span_id}", cancelBatchNotificationHandler.Handle).Methods(http.MethodDelete)

	// Создаем HTTP сервер
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Запускаем HTTP сервер
	go func() {
		log.Info("Starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// КРИТИЧНО: Останавливаем Worker ПЕРЕД сервером
	processor.Stop()
	scheduler.Stop()
	log.Info("Worker components stopped")

	// Останавливаем сбор метрик
	if cfg.Metrics.Enabled {
		close(stopMetricsCh)
		log.Info("Metrics collection stopped")
	}

	// Graceful shutdown HTTP сервера
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.Server.ShutdownTimeout)*time.Second,
	)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped gracefully")
}
