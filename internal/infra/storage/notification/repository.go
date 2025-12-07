package notification

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/m04kA/SMC-NotificationService/internal/domain"
	"github.com/m04kA/SMC-NotificationService/pkg/dbmetrics"
	"github.com/m04kA/SMC-NotificationService/pkg/psqlbuilder"
)

// Repository репозиторий для работы с уведомлениями
type Repository struct {
	db DBExecutor
}

// NewRepository создает новый экземпляр репозитория уведомлений
func NewRepository(db DBExecutor) *Repository {
	return &Repository{db: db}
}

// Create создает одно уведомление и возвращает его ID
func (r *Repository) Create(ctx context.Context, notification *domain.Notification) (int64, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Insert("notifications").
		Columns(
			"telegram_user_id",
			"chat_id",
			"span_id",
			"message_text",
			"image_urls",
			"inline_buttons",
			"notification_type",
			"status",
			"scheduled_for",
			"metadata",
		).
		Values(
			notification.TelegramUserID,
			notification.ChatID,
			notification.SpanID,
			notification.MessageText,
			pq.Array(notification.ImageURLs),
			notification.InlineButtons,
			notification.Type,
			notification.Status,
			notification.ScheduledFor,
			notification.Metadata,
		).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()

	if err != nil {
		return 0, fmt.Errorf("%w: Create - build insert query: %v", ErrBuildQuery, err)
	}

	var id int64
	var createdAt, updatedAt sql.NullTime
	err = executor.QueryRowContext(ctx, query, args...).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return 0, fmt.Errorf("%w: Create - execute insert: %v", ErrExecQuery, err)
	}

	notification.ID = id
	notification.CreatedAt = createdAt.Time
	notification.UpdatedAt = updatedAt.Time

	return id, nil
}

// CreateBatch создает несколько уведомлений за один запрос (для массовых рассылок)
// Возвращает список ID созданных записей
func (r *Repository) CreateBatch(ctx context.Context, notifications []*domain.Notification) ([]int64, error) {
	if len(notifications) == 0 {
		return []int64{}, nil
	}

	executor := dbmetrics.GetExecutor(ctx, r.db)

	builder := psqlbuilder.Insert("notifications").
		Columns(
			"telegram_user_id",
			"chat_id",
			"span_id",
			"message_text",
			"image_urls",
			"inline_buttons",
			"notification_type",
			"status",
			"scheduled_for",
			"metadata",
		)

	// Добавляем все уведомления в один batch запрос
	for _, n := range notifications {
		builder = builder.Values(
			n.TelegramUserID,
			n.ChatID,
			n.SpanID,
			n.MessageText,
			pq.Array(n.ImageURLs),
			n.InlineButtons,
			n.Type,
			n.Status,
			n.ScheduledFor,
			n.Metadata,
		)
	}

	query, args, err := builder.Suffix("RETURNING id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: CreateBatch - build insert query: %v", ErrBuildQuery, err)
	}

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: CreateBatch - execute insert: %v", ErrExecQuery, err)
	}
	defer rows.Close()

	ids := make([]int64, 0, len(notifications))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%w: CreateBatch - scan id: %v", ErrScanRow, err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: CreateBatch - rows error: %v", ErrScanRow, err)
	}

	return ids, nil
}

// GetByID получает уведомление по ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*domain.Notification, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Select(
		"id",
		"telegram_user_id",
		"chat_id",
		"span_id",
		"message_text",
		"image_urls",
		"inline_buttons",
		"notification_type",
		"status",
		"scheduled_for",
		"sent_at",
		"metadata",
		"error_message",
		"retry_count",
		"created_at",
		"updated_at",
	).
		From("notifications").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: GetByID - build select query: %v", ErrBuildQuery, err)
	}

	var notification domain.Notification
	var createdAt, updatedAt sql.NullTime

	err = executor.QueryRowContext(ctx, query, args...).Scan(
		&notification.ID,
		&notification.TelegramUserID,
		&notification.ChatID,
		&notification.SpanID,
		&notification.MessageText,
		pq.Array(&notification.ImageURLs),
		&notification.InlineButtons,
		&notification.Type,
		&notification.Status,
		&notification.ScheduledFor,
		&notification.SentAt,
		&notification.Metadata,
		&notification.ErrorMessage,
		&notification.RetryCount,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotificationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: GetByID - scan notification: %v", ErrScanRow, err)
	}

	notification.CreatedAt = createdAt.Time
	notification.UpdatedAt = updatedAt.Time

	return &notification, nil
}

// GetBySpanID получает все уведомления по span_id (массовая рассылка)
func (r *Repository) GetBySpanID(ctx context.Context, spanID string) ([]*domain.Notification, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Select(
		"id",
		"telegram_user_id",
		"chat_id",
		"span_id",
		"message_text",
		"image_urls",
		"inline_buttons",
		"notification_type",
		"status",
		"scheduled_for",
		"sent_at",
		"metadata",
		"error_message",
		"retry_count",
		"created_at",
		"updated_at",
	).
		From("notifications").
		Where(squirrel.Eq{"span_id": spanID}).
		OrderBy("created_at ASC").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: GetBySpanID - build select query: %v", ErrBuildQuery, err)
	}

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: GetBySpanID - execute query: %v", ErrExecQuery, err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// List получает список уведомлений с фильтрацией
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*domain.Notification, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	selectBuilder := psqlbuilder.Select(
		"id",
		"telegram_user_id",
		"chat_id",
		"span_id",
		"message_text",
		"image_urls",
		"inline_buttons",
		"notification_type",
		"status",
		"scheduled_for",
		"sent_at",
		"metadata",
		"error_message",
		"retry_count",
		"created_at",
		"updated_at",
	).
		From("notifications").
		OrderBy("created_at DESC")

	// Фильтрация по статусу
	if filter.Status != nil {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"status": *filter.Status})
	}

	// Фильтрация по типу
	if filter.Type != nil {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"notification_type": *filter.Type})
	}

	// Фильтрация по telegram_user_id
	if filter.TelegramUserID != nil {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"telegram_user_id": *filter.TelegramUserID})
	}

	// Фильтрация по span_id
	if filter.SpanID != nil {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"span_id": *filter.SpanID})
	}

	// Пагинация
	if filter.Limit > 0 {
		selectBuilder = selectBuilder.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		selectBuilder = selectBuilder.Offset(uint64(filter.Offset))
	}

	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: List - build select query: %v", ErrBuildQuery, err)
	}

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: List - execute query: %v", ErrExecQuery, err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// scanNotifications сканирует результаты запроса в слайс уведомлений
func (r *Repository) scanNotifications(rows *sql.Rows) ([]*domain.Notification, error) {
	notifications := make([]*domain.Notification, 0)

	for rows.Next() {
		var notification domain.Notification
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&notification.ID,
			&notification.TelegramUserID,
			&notification.ChatID,
			&notification.SpanID,
			&notification.MessageText,
			pq.Array(&notification.ImageURLs),
			&notification.InlineButtons,
			&notification.Type,
			&notification.Status,
			&notification.ScheduledFor,
			&notification.SentAt,
			&notification.Metadata,
			&notification.ErrorMessage,
			&notification.RetryCount,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("%w: scanNotifications - scan row: %v", ErrScanRow, err)
		}

		notification.CreatedAt = createdAt.Time
		notification.UpdatedAt = updatedAt.Time

		notifications = append(notifications, &notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: scanNotifications - rows error: %v", ErrScanRow, err)
	}

	return notifications, nil
}
