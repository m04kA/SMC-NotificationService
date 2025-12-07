package dbmetrics

import (
	"context"
	"database/sql"
)

// DBExecutor интерфейс для выполнения SQL запросов
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// TxExecutor интерфейс для выполнения запросов в транзакции
type TxExecutor interface {
	DBExecutor
	Commit() error
	Rollback() error
}

// SqlTxWrapper обёртка для *sql.Tx чтобы реализовать TxExecutor
type SqlTxWrapper struct {
	*sql.Tx
}

func (w *SqlTxWrapper) Commit() error {
	return w.Tx.Commit()
}

func (w *SqlTxWrapper) Rollback() error {
	return w.Tx.Rollback()
}
