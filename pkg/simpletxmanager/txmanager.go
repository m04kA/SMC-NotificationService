package simpletxmanager

import (
	"context"
	"database/sql"
	"fmt"
)

// TransactionManager простой менеджер транзакций без метрик
type TransactionManager struct {
	db *sql.DB
}

// NewTransactionManager создаёт новый менеджер транзакций
func NewTransactionManager(db *sql.DB) *TransactionManager {
	return &TransactionManager{
		db: db,
	}
}

// DoSerializable выполняет функцию внутри транзакции с уровнем изоляции Serializable
// Если функция завершается без ошибки, транзакция фиксируется (commit)
// Если функция возвращает ошибку, транзакция откатывается (rollback)
func (tm *TransactionManager) DoSerializable(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.DoWithOptions(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	}, fn)
}

// DoWithOptions выполняет функцию внутри транзакции с указанными опциями
func (tm *TransactionManager) DoWithOptions(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error {
	// Начинаем новую транзакцию
	tx, err := tm.db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer для обработки паники и автоматического rollback
	defer func() {
		if p := recover(); p != nil {
			// При панике откатываем транзакцию
			_ = tx.Rollback()
			panic(p) // Пробрасываем панику дальше
		}
	}()

	// Выполняем функцию внутри транзакции
	// Передаём контекст, который использует транзакцию
	fnErr := fn(ctx)

	if fnErr != nil {
		// При ошибке откатываем транзакцию
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction rollback failed: %w (original error: %v)", rbErr, fnErr)
		}
		return fnErr
	}

	// Фиксируем транзакцию
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("failed to commit transaction: %w", commitErr)
	}

	return nil
}
