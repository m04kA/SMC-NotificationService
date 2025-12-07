package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/m04kA/SMC-NotificationService/internal/domain"
	"github.com/m04kA/SMC-NotificationService/pkg/dbmetrics"
	"github.com/m04kA/SMC-NotificationService/pkg/psqlbuilder"
)

// GetPendingNotifications получает список pending уведомлений для немедленной отправки
// Используется processor'ом для обработки очереди
func (r *Repository) GetPendingNotifications(ctx context.Context, limit int) ([]*domain.Notification, error) {
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
		Where(squirrel.Eq{"status": domain.NotificationStatusPending}).
		OrderBy("created_at ASC").
		Limit(uint64(limit)).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: GetPendingNotifications - build select query: %v", ErrBuildQuery, err)
	}

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: GetPendingNotifications - execute query: %v", ErrExecQuery, err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// GetScheduledNotifications получает список scheduled уведомлений для загрузки в gocron
// Используется scheduler'ом при старте приложения
func (r *Repository) GetScheduledNotifications(ctx context.Context) ([]*domain.Notification, error) {
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
		Where(squirrel.Eq{"status": domain.NotificationStatusScheduled}).
		OrderBy("scheduled_for ASC").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: GetScheduledNotifications - build select query: %v", ErrBuildQuery, err)
	}

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: GetScheduledNotifications - execute query: %v", ErrExecQuery, err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// UpdateStatus обновляет статус уведомления
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status domain.NotificationStatus) error {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Update("notifications").
		Set("status", status).
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return fmt.Errorf("%w: UpdateStatus - build update query: %v", ErrBuildQuery, err)
	}

	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: UpdateStatus - execute update: %v", ErrExecQuery, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: UpdateStatus - get rows affected: %v", ErrExecQuery, err)
	}

	if rowsAffected == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// MarkAsSent помечает уведомление как успешно отправленное
func (r *Repository) MarkAsSent(ctx context.Context, id int64, sentAt time.Time) error {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Update("notifications").
		Set("status", domain.NotificationStatusSent).
		Set("sent_at", sentAt).
		Set("error_message", nil).
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return fmt.Errorf("%w: MarkAsSent - build update query: %v", ErrBuildQuery, err)
	}

	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: MarkAsSent - execute update: %v", ErrExecQuery, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: MarkAsSent - get rows affected: %v", ErrExecQuery, err)
	}

	if rowsAffected == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// MarkAsFailed помечает уведомление как неудачное с сообщением об ошибке
// Опционально увеличивает счётчик попыток отправки
func (r *Repository) MarkAsFailed(ctx context.Context, id int64, errorMsg string, incrementRetry bool) error {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	updateBuilder := psqlbuilder.Update("notifications").
		Set("status", domain.NotificationStatusFailed).
		Set("error_message", errorMsg)

	// Если нужно увеличить счётчик попыток
	if incrementRetry {
		updateBuilder = updateBuilder.Set("retry_count", squirrel.Expr("retry_count + 1"))
	}

	query, args, err := updateBuilder.Where(squirrel.Eq{"id": id}).ToSql()

	if err != nil {
		return fmt.Errorf("%w: MarkAsFailed - build update query: %v", ErrBuildQuery, err)
	}

	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: MarkAsFailed - execute update: %v", ErrExecQuery, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: MarkAsFailed - get rows affected: %v", ErrExecQuery, err)
	}

	if rowsAffected == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// Cancel отменяет отложенное уведомление по ID
// Может быть отменено только pending или scheduled уведомление
func (r *Repository) Cancel(ctx context.Context, id int64) error {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Update("notifications").
		Set("status", domain.NotificationStatusCancelled).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Or{
			squirrel.Eq{"status": domain.NotificationStatusPending},
			squirrel.Eq{"status": domain.NotificationStatusScheduled},
		}).
		ToSql()

	if err != nil {
		return fmt.Errorf("%w: Cancel - build update query: %v", ErrBuildQuery, err)
	}

	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: Cancel - execute update: %v", ErrExecQuery, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: Cancel - get rows affected: %v", ErrExecQuery, err)
	}

	if rowsAffected == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// CancelBySpanID отменяет все уведомления массовой рассылки по span_id
// Может быть отменено только pending или scheduled уведомления
// Возвращает количество отмененных уведомлений
func (r *Repository) CancelBySpanID(ctx context.Context, spanID string) (int, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Update("notifications").
		Set("status", domain.NotificationStatusCancelled).
		Where(squirrel.Eq{"span_id": spanID}).
		Where(squirrel.Or{
			squirrel.Eq{"status": domain.NotificationStatusPending},
			squirrel.Eq{"status": domain.NotificationStatusScheduled},
		}).
		ToSql()

	if err != nil {
		return 0, fmt.Errorf("%w: CancelBySpanID - build update query: %v", ErrBuildQuery, err)
	}

	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("%w: CancelBySpanID - execute update: %v", ErrExecQuery, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%w: CancelBySpanID - get rows affected: %v", ErrExecQuery, err)
	}

	return int(rowsAffected), nil
}

// IncrementRetryCount увеличивает счётчик попыток отправки
func (r *Repository) IncrementRetryCount(ctx context.Context, id int64) error {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Update("notifications").
		Set("retry_count", squirrel.Expr("retry_count + 1")).
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return fmt.Errorf("%w: IncrementRetryCount - build update query: %v", ErrBuildQuery, err)
	}

	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: IncrementRetryCount - execute update: %v", ErrExecQuery, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: IncrementRetryCount - get rows affected: %v", ErrExecQuery, err)
	}

	if rowsAffected == 0 {
		return ErrNotificationNotFound
	}

	return nil
}
