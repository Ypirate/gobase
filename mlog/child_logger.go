package mlog

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a child logger with fixed fields.
type Logger struct {
	fields []zap.Field
}

// With creates a child logger with fixed fields that will be included in all log entries.
func With(fields ...zap.Field) *Logger {
	return &Logger{fields: fields}
}

func (l *Logger) Debugf(ctx context.Context, format string, args ...interface{}) {
	logChildWithFields(ctx, zapcore.DebugLevel, l.fields, format, args...)
}

func (l *Logger) Infof(ctx context.Context, format string, args ...interface{}) {
	logChildWithFields(ctx, zapcore.InfoLevel, l.fields, format, args...)
}

func (l *Logger) Warnf(ctx context.Context, format string, args ...interface{}) {
	logChildWithFields(ctx, zapcore.WarnLevel, l.fields, format, args...)
}

func (l *Logger) Errorf(ctx context.Context, format string, args ...interface{}) {
	logChildWithFields(ctx, zapcore.ErrorLevel, l.fields, format, args...)
}

func (l *Logger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	logChildWithFields(ctx, zapcore.FatalLevel, l.fields, format, args...)
}

func (l *Logger) Panicf(ctx context.Context, format string, args ...interface{}) {
	logChildWithFields(ctx, zapcore.PanicLevel, l.fields, format, args...)
}

// logChildWithFields logs with both fixed fields and context fields (for child logger).
func logChildWithFields(ctx context.Context, level zapcore.Level, fixedFields []zap.Field, format string, args ...interface{}) {
	lg := getLogger()
	if lg == nil {
		fmt.Fprintf(os.Stderr, "[UNINIT] %s\n", fmt.Sprintf(format, args...))
		return
	}

	msg := fmt.Sprintf(format, args...)
	msg = truncateUTF8(msg, maxLogMessageLength)

	// Merge fixed fields with context fields
	fields := append(fixedFields, extractFieldsFromContext(ctx)...)

	// Use AddCallerSkip(1) to skip this wrapper function
	lg.WithOptions(zap.AddCallerSkip(1)).Log(level, msg, fields...)
}
