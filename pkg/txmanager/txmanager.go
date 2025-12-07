package txmanager

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/m04kA/SMC-BookingService/pkg/dbmetrics"
)

// TransactionManager управляет транзакциями с поддержкой метрик через dbmetrics.DB
// Совместим с существующим механизмом передачи транзакций через контекст
type TransactionManager struct {
	db *dbmetrics.DB
}

// NewTransactionManager создаёт новый менеджер транзакций
func NewTransactionManager(db *dbmetrics.DB) *TransactionManager {
	return &TransactionManager{
		db: db,
	}
}

// Do выполняет функцию внутри транзакции
// Если функция завершается без ошибки, транзакция фиксируется (commit)
// Если функция возвращает ошибку, транзакция откатывается (rollback)
//
// Пример использования:
//   err := tm.Do(ctx, func(ctx context.Context) error {
//       // Все операции внутри этой функции будут выполнены в одной транзакции
//       booking, err := s.bookingRepo.Create(ctx, booking)
//       if err != nil {
//           return err
//       }
//       return s.notifyUser(ctx, booking.UserID)
//   })
func (tm *TransactionManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.DoWithOptions(ctx, nil, fn)
}

// DoWithOptions выполняет функцию внутри транзакции с указанными опциями
// Позволяет настроить уровень изоляции транзакции
//
// Пример использования:
//   err := tm.DoWithOptions(ctx, &sql.TxOptions{
//       Isolation: sql.LevelSerializable,
//   }, func(ctx context.Context) error {
//       // Операции с высоким уровнем изоляции
//       return s.bookingRepo.Create(ctx, booking)
//   })
func (tm *TransactionManager) DoWithOptions(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error {
	// Проверяем, не находимся ли мы уже в транзакции
	if dbmetrics.IsInTransaction(ctx) {
		// Если уже в транзакции, просто выполняем функцию
		// (поддержка вложенных транзакций через повторное использование существующей)
		return fn(ctx)
	}

	// Начинаем новую транзакцию через dbmetrics.DB (с метриками)
	tx, err := tm.db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Добавляем транзакцию в контекст
	txCtx := dbmetrics.WithTx(ctx, tx)

	// Defer для обработки паники и автоматического rollback
	defer func() {
		if p := recover(); p != nil {
			// При панике откатываем транзакцию
			_ = tx.Rollback()
			panic(p) // Пробрасываем панику дальше
		}
	}()

	// Выполняем функцию внутри транзакции
	fnErr := fn(txCtx)

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

// DoReadOnly выполняет функцию внутри read-only транзакции
// Оптимизация для операций только на чтение
func (tm *TransactionManager) DoReadOnly(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.DoWithOptions(ctx, &sql.TxOptions{
		ReadOnly: true,
	}, fn)
}

// DoSerializable выполняет функцию внутри транзакции с уровнем изоляции Serializable
// Используется для критичных операций, требующих полной изоляции
//
// Пример: создание бронирования с проверкой доступности слота
func (tm *TransactionManager) DoSerializable(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.DoWithOptions(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	}, fn)
}
