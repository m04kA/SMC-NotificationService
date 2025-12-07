package notification

import "errors"

var (
	// ErrNotificationNotFound возвращается, когда уведомление не найдено
	ErrNotificationNotFound = errors.New("repository: notification not found")

	// ErrBuildQuery возвращается при ошибке построения SQL запроса
	ErrBuildQuery = errors.New("repository: failed to build SQL query")

	// ErrExecQuery возвращается при ошибке выполнения SQL запроса
	ErrExecQuery = errors.New("repository: failed to execute SQL query")

	// ErrScanRow возвращается при ошибке сканирования строки результата
	ErrScanRow = errors.New("repository: failed to scan row")

	// ErrBeginTx возвращается при ошибке начала транзакции
	ErrBeginTx = errors.New("repository: failed to begin transaction")

	// ErrCommitTx возвращается при ошибке коммита транзакции
	ErrCommitTx = errors.New("repository: failed to commit transaction")

	// ErrRollbackTx возвращается при ошибке отката транзакции
	ErrRollbackTx = errors.New("repository: failed to rollback transaction")
)
