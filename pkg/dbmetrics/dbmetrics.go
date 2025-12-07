package dbmetrics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/m04kA/SMC-NotificationService/pkg/metrics"
)

// DB обёртка над *sql.DB с поддержкой метрик
type DB struct {
	*sql.DB
	metrics     *metrics.Metrics
	serviceName string
}

// Wrap оборачивает *sql.DB для сбора метрик
func Wrap(db *sql.DB, metrics *metrics.Metrics, serviceName string) *DB {
	return &DB{
		DB:          db,
		metrics:     metrics,
		serviceName: serviceName,
	}
}

// QueryContext выполняет запрос с контекстом и сбором метрик
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	operation, table := parseQuery(query)

	rows, err := db.DB.QueryContext(ctx, query, args...)

	duration := time.Since(start).Seconds()

	if err != nil {
		db.metrics.RecordDBQuery(db.serviceName, operation, table, "error", duration)
		db.metrics.RecordDBError(db.serviceName, operation, table, categorizeDBError(err))
		return nil, err
	}

	db.metrics.RecordDBQuery(db.serviceName, operation, table, "success", duration)
	return rows, nil
}

// QueryRowContext выполняет запрос одной строки с контекстом и сбором метрик
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	operation, table := parseQuery(query)

	row := db.DB.QueryRowContext(ctx, query, args...)

	duration := time.Since(start).Seconds()

	// Для QueryRow успех определяется при Scan(), поэтому записываем только время
	db.metrics.RecordDBQuery(db.serviceName, operation, table, "success", duration)

	return row
}

// ExecContext выполняет команду с контекстом и сбором метрик
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	operation, table := parseQuery(query)

	result, err := db.DB.ExecContext(ctx, query, args...)

	duration := time.Since(start).Seconds()

	if err != nil {
		db.metrics.RecordDBQuery(db.serviceName, operation, table, "error", duration)
		db.metrics.RecordDBError(db.serviceName, operation, table, categorizeDBError(err))
		return nil, err
	}

	db.metrics.RecordDBQuery(db.serviceName, operation, table, "success", duration)
	return result, nil
}

// BeginTx начинает транзакцию с контекстом и сбором метрик
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (TxExecutor, error) {
	start := time.Now()

	tx, err := db.DB.BeginTx(ctx, opts)

	duration := time.Since(start).Seconds()

	if err != nil {
		db.metrics.RecordDBQuery(db.serviceName, "begin_tx", "transaction", "error", duration)
		db.metrics.RecordDBError(db.serviceName, "begin_tx", "transaction", categorizeDBError(err))
		return nil, err
	}

	db.metrics.RecordDBQuery(db.serviceName, "begin_tx", "transaction", "success", duration)

	return &Tx{
		Tx:          tx,
		metrics:     db.metrics,
		serviceName: db.serviceName,
	}, nil
}

// UpdateConnectionStats обновляет метрики connection pool
func (db *DB) UpdateConnectionStats() {
	stats := db.DB.Stats()
	db.metrics.UpdateDBConnectionStats(
		stats.InUse,
		stats.Idle,
		stats.MaxOpenConnections,
	)
}

// Tx обёртка над *sql.Tx с поддержкой метрик
type Tx struct {
	*sql.Tx
	metrics     *metrics.Metrics
	serviceName string
}

// QueryContext выполняет запрос в транзакции с контекстом и сбором метрик
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	operation, table := parseQuery(query)

	rows, err := tx.Tx.QueryContext(ctx, query, args...)

	duration := time.Since(start).Seconds()

	if err != nil {
		tx.metrics.RecordDBQuery(tx.serviceName, operation, table, "error", duration)
		tx.metrics.RecordDBError(tx.serviceName, operation, table, categorizeDBError(err))
		return nil, err
	}

	tx.metrics.RecordDBQuery(tx.serviceName, operation, table, "success", duration)
	return rows, nil
}

// QueryRowContext выполняет запрос одной строки в транзакции
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	operation, table := parseQuery(query)

	row := tx.Tx.QueryRowContext(ctx, query, args...)

	duration := time.Since(start).Seconds()

	tx.metrics.RecordDBQuery(tx.serviceName, operation, table, "success", duration)

	return row
}

// ExecContext выполняет команду в транзакции с контекстом и сбором метрик
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	operation, table := parseQuery(query)

	result, err := tx.Tx.ExecContext(ctx, query, args...)

	duration := time.Since(start).Seconds()

	if err != nil {
		tx.metrics.RecordDBQuery(tx.serviceName, operation, table, "error", duration)
		tx.metrics.RecordDBError(tx.serviceName, operation, table, categorizeDBError(err))
		return nil, err
	}

	tx.metrics.RecordDBQuery(tx.serviceName, operation, table, "success", duration)
	return result, nil
}

// Commit фиксирует транзакцию с метриками
func (tx *Tx) Commit() error {
	start := time.Now()

	err := tx.Tx.Commit()

	duration := time.Since(start).Seconds()

	if err != nil {
		tx.metrics.RecordDBQuery(tx.serviceName, "commit", "transaction", "error", duration)
		tx.metrics.RecordDBError(tx.serviceName, "commit", "transaction", categorizeDBError(err))
		return err
	}

	tx.metrics.RecordDBQuery(tx.serviceName, "commit", "transaction", "success", duration)
	return nil
}

// Rollback откатывает транзакцию с метриками
func (tx *Tx) Rollback() error {
	start := time.Now()

	err := tx.Tx.Rollback()

	duration := time.Since(start).Seconds()

	if err != nil {
		tx.metrics.RecordDBQuery(tx.serviceName, "rollback", "transaction", "error", duration)
		tx.metrics.RecordDBError(tx.serviceName, "rollback", "transaction", categorizeDBError(err))
		return err
	}

	tx.metrics.RecordDBQuery(tx.serviceName, "rollback", "transaction", "success", duration)
	return nil
}

// parseQuery извлекает тип операции и имя таблицы из SQL запроса
func parseQuery(query string) (operation, table string) {
	query = strings.TrimSpace(strings.ToUpper(query))

	// Определяем операцию
	switch {
	case strings.HasPrefix(query, "SELECT"):
		operation = "select"
	case strings.HasPrefix(query, "INSERT"):
		operation = "insert"
	case strings.HasPrefix(query, "UPDATE"):
		operation = "update"
	case strings.HasPrefix(query, "DELETE"):
		operation = "delete"
	default:
		operation = "other"
	}

	// Пытаемся извлечь имя таблицы
	table = extractTableName(query, operation)

	return operation, table
}

// extractTableName извлекает имя таблицы из SQL запроса
func extractTableName(query, operation string) string {
	words := strings.Fields(query)

	switch operation {
	case "select":
		// SELECT ... FROM table_name
		for i, word := range words {
			if word == "FROM" && i+1 < len(words) {
				return cleanTableName(words[i+1])
			}
		}
	case "insert":
		// INSERT INTO table_name
		for i, word := range words {
			if word == "INTO" && i+1 < len(words) {
				return cleanTableName(words[i+1])
			}
		}
	case "update":
		// UPDATE table_name
		if len(words) >= 2 {
			return cleanTableName(words[1])
		}
	case "delete":
		// DELETE FROM table_name
		for i, word := range words {
			if word == "FROM" && i+1 < len(words) {
				return cleanTableName(words[i+1])
			}
		}
	}

	return "unknown"
}

// cleanTableName очищает имя таблицы от лишних символов
func cleanTableName(name string) string {
	// Убираем кавычки и скобки
	name = strings.Trim(name, `"'()`)
	// Берём только имя таблицы (без схемы)
	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return name
}

// categorizeDBError категоризирует ошибки базы данных
func categorizeDBError(err error) string {
	if err == nil {
		return "none"
	}

	errStr := err.Error()

	switch {
	case err == sql.ErrNoRows:
		return "no_rows"
	case err == sql.ErrTxDone:
		return "transaction_done"
	case err == sql.ErrConnDone:
		return "connection_done"
	case strings.Contains(errStr, "duplicate key"):
		return "duplicate_key"
	case strings.Contains(errStr, "foreign key"):
		return "foreign_key_violation"
	case strings.Contains(errStr, "not null"):
		return "not_null_violation"
	case strings.Contains(errStr, "check constraint"):
		return "check_constraint_violation"
	case strings.Contains(errStr, "connection refused"):
		return "connection_refused"
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "deadlock"):
		return "deadlock"
	default:
		return "unknown"
	}
}

// Убедимся что DB и Tx реализуют DBExecutor и TxExecutor
var (
	_ DBExecutor = (*DB)(nil)
	_ DBExecutor = (*Tx)(nil)
	_ TxExecutor = (*Tx)(nil)
)

// StartConnectionStatsCollector запускает фоновую горутину для сбора метрик connection pool
func (db *DB) StartConnectionStatsCollector(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			db.UpdateConnectionStats()
		case <-stopCh:
			return
		}
	}
}

// WrapWithDefault создаёт обёртку с дефолтным сборщиком метрик connection pool (каждые 15 секунд)
func WrapWithDefault(db *sql.DB, metrics *metrics.Metrics, serviceName string, stopCh <-chan struct{}) *DB {
	wrapped := Wrap(db, metrics, serviceName)
	go wrapped.StartConnectionStatsCollector(15*time.Second, stopCh)
	return wrapped
}

// Helper для совместимости с существующим кодом
// Преобразует DBExecutor обратно в стандартные типы для legacy кода
func Unwrap(executor DBExecutor) interface{} {
	switch v := executor.(type) {
	case *DB:
		return v.DB
	case *Tx:
		return v.Tx
	default:
		return executor
	}
}

// PrintQueryStats выводит статистику запросов (для debugging)
func (db *DB) PrintQueryStats() {
	stats := db.DB.Stats()
	fmt.Printf("DB Stats:\n")
	fmt.Printf("  Open Connections: %d\n", stats.OpenConnections)
	fmt.Printf("  In Use: %d\n", stats.InUse)
	fmt.Printf("  Idle: %d\n", stats.Idle)
	fmt.Printf("  Wait Count: %d\n", stats.WaitCount)
	fmt.Printf("  Wait Duration: %s\n", stats.WaitDuration)
	fmt.Printf("  Max Idle Closed: %d\n", stats.MaxIdleClosed)
	fmt.Printf("  Max Lifetime Closed: %d\n", stats.MaxLifetimeClosed)
}

// Transaction context helpers

// txContextKey тип для ключа транзакции в контексте
type txContextKey struct{}

// WithTx возвращает новый контекст с транзакцией
// Используется для передачи транзакции через контекст в методы репозитория
func WithTx(ctx context.Context, tx TxExecutor) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// GetExecutor возвращает executor из контекста (если есть транзакция) или переданный db
// Позволяет методам репозитория работать как с транзакцией, так и без неё
func GetExecutor(ctx context.Context, db DBExecutor) DBExecutor {
	if tx, ok := ctx.Value(txContextKey{}).(TxExecutor); ok && tx != nil {
		return tx
	}
	return db
}

// IsInTransaction проверяет, выполняется ли запрос в контексте транзакции
func IsInTransaction(ctx context.Context) bool {
	_, ok := ctx.Value(txContextKey{}).(TxExecutor)
	return ok
}
