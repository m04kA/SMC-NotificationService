package logger

import (
	"io"
	"log"
	"os"
)

type Logger struct {
	info    *log.Logger
	warn    *log.Logger
	err     *log.Logger
	logFile *os.File
}

// New создает новый экземпляр логгера с записью в консоль и файл
func New(logFilePath string) (*Logger, error) {
	// Открываем файл для логов
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	// Настраиваем мультиплексор для warning и error
	warnErrorWriter := io.MultiWriter(os.Stdout, logFile)

	return &Logger{
		info:    log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lshortfile),
		warn:    log.New(warnErrorWriter, "[WARN] ", log.LstdFlags|log.Lshortfile),
		err:     log.New(warnErrorWriter, "[ERROR] ", log.LstdFlags|log.Lshortfile),
		logFile: logFile,
	}, nil
}

// Close закрывает файл логов
func (l *Logger) Close() error {
	if l != nil && l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Info логирует информационные сообщения (только консоль)
func (l *Logger) Info(format string, v ...interface{}) {
	if l != nil {
		l.info.Printf(format, v...)
	}
}

// Warn логирует предупреждения (консоль + файл)
func (l *Logger) Warn(format string, v ...interface{}) {
	if l != nil {
		l.warn.Printf(format, v...)
	}
}

// Error логирует ошибки (консоль + файл)
func (l *Logger) Error(format string, v ...interface{}) {
	if l != nil {
		l.err.Printf(format, v...)
	}
}

// Fatal логирует критическую ошибку и завершает программу
func (l *Logger) Fatal(format string, v ...interface{}) {
	if l != nil {
		l.err.Fatalf(format, v...)
	}
}
