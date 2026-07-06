package logger

import (
	"log/slog"
	"os"
)

type Logger struct {
	log *slog.Logger
}

func New() *Logger {
	return &Logger{
		log: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}
}

func (l *Logger) Info(msg string, args ...any) {
	l.log.Info(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.log.Error(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.log.Warn(msg, args...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.log.Debug(msg, args...)
}
