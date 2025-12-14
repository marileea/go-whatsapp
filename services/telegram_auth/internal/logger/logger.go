package logger

import (
    "context"
    "log/slog"
    "os"
    "time"
)

// Logger is a wrapper around slog.Logger
type Logger struct {
    *slog.Logger
}

// New creates a new logger with the specified level
func New(level, format string) *Logger {
    var logLevel slog.Level
    switch level {
    case "debug":
        logLevel = slog.LevelDebug
    case "info":
        logLevel = slog.LevelInfo
    case "warn":
        logLevel = slog.LevelWarn
    case "error":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }

    handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: logLevel,
    })

    return &Logger{
        Logger: slog.New(handler),
    }
}

// With creates a logger with additional fields
func (l *Logger) With(fields ...any) *Logger {
    return &Logger{
        Logger: l.Logger.With(fields...),
    }
}

// Info logs an info message
func (l *Logger) Info(ctx context.Context, msg string, fields ...any) {
    l.Log(ctx, slog.LevelInfo, msg, fields...)
}

// Debug logs a debug message
func (l *Logger) Debug(ctx context.Context, msg string, fields ...any) {
    l.Log(ctx, slog.LevelDebug, msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(ctx context.Context, msg string, fields ...any) {
    l.Log(ctx, slog.LevelWarn, msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(ctx context.Context, msg string, fields ...any) {
    l.Log(ctx, slog.LevelError, msg, fields...)
}

// ErrorWithErr logs an error message with an error
func (l *Logger) ErrorWithErr(ctx context.Context, msg string, err error, fields ...any) {
    fields = append(fields, "error", err)
    l.Error(ctx, msg, fields...)
}

// Log logs a message with the specified level
func (l *Logger) Log(ctx context.Context, level slog.Level, msg string, fields ...any) {
    if len(fields)%2 != 0 {
        fields = append(fields, "unpaired_key", "value")
    }
    
    record := slog.NewRecord(time.Now(), level, msg, 0)
    if requestID := ctx.Value("request_id"); requestID != nil {
        record.Add(requestID)
    }
    record.Add(fields...)
    
    l.Handler().Handle(context.Background(), record)
}