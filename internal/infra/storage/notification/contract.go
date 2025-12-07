package notification

import (
	"github.com/m04kA/SMC-NotificationService/pkg/dbmetrics"
)

// Переиспользуем интерфейсы из dbmetrics для работы с БД
type DBExecutor = dbmetrics.DBExecutor
type TxExecutor = dbmetrics.TxExecutor
