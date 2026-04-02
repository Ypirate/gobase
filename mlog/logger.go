package mlog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Errorf(ctx context.Context, format string, args ...interface{}) {
	logWithCtx(ctx, zapcore.ErrorLevel, format, args...)
}

func Fatalf(ctx context.Context, format string, args ...interface{}) {
	logWithCtx(ctx, zapcore.FatalLevel, format, args...)
}

func Panicf(ctx context.Context, format string, args ...interface{}) {
	logWithCtx(ctx, zapcore.PanicLevel, format, args...)
}

func Warnf(ctx context.Context, format string, args ...interface{}) {
	logWithCtx(ctx, zapcore.WarnLevel, format, args...)
}

func Infof(ctx context.Context, format string, args ...interface{}) {
	logWithCtx(ctx, zapcore.InfoLevel, format, args...)
}

func Debugf(ctx context.Context, format string, args ...interface{}) {
	logWithCtx(ctx, zapcore.DebugLevel, format, args...)
}

// Info logs with structured fields (no formatting).
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	logWithFields(ctx, zapcore.InfoLevel, nil, msg, fields...)
}

// Error logs with structured fields (no formatting).
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	logWithFields(ctx, zapcore.ErrorLevel, nil, msg, fields...)
}

// Warn logs with structured fields (no formatting).
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	logWithFields(ctx, zapcore.WarnLevel, nil, msg, fields...)
}

// Debug logs with structured fields (no formatting).
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	logWithFields(ctx, zapcore.DebugLevel, nil, msg, fields...)
}

func logWithCtx(ctx context.Context, level zapcore.Level, format string, args ...interface{}) {
	l := getLogger()
	if l == nil {
		// fallback: logger not initialized
		fmt.Fprintf(os.Stderr, "[UNINIT] %s\n", fmt.Sprintf(format, args...))
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = truncateUTF8(msg, maxLogMessageLength)

	fields := extractFieldsFromContext(ctx)

	switch level {
	case zapcore.DebugLevel:
		l.Debug(msg, fields...)
	case zapcore.InfoLevel:
		l.Info(msg, fields...)
	case zapcore.WarnLevel:
		l.Warn(msg, fields...)
	case zapcore.ErrorLevel:
		l.Error(msg, fields...)
	case zapcore.PanicLevel:
		l.Panic(msg, fields...)
	case zapcore.FatalLevel:
		l.Fatal(msg, fields...)
	default:
		panic("logger level unhandled default case")
	}
}

func extractFieldsFromContext(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}
	return getFieldsFromContext(ctx)
}

func parseLevel(levelStr string) zapcore.Level {
	levelStr = strings.ToLower(strings.TrimSpace(levelStr))
	switch levelStr {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	// 找到不超过 maxBytes 的最大 rune 边界
	for i := maxBytes; i >= 0; i-- {
		if utf8.RuneStart(s[i]) {
			return s[:i] + " ... [truncated]"
		}
	}
	return s[:maxBytes] + " ... [truncated]" // fallback
}

// logWithFields logs with structured fields (used by Info/Error/Warn/Debug and child logger).
func logWithFields(ctx context.Context, level zapcore.Level, fixedFields []zap.Field, msg string, additionalFields ...zap.Field) {
	lg := getLogger()
	if lg == nil {
		fmt.Fprintf(os.Stderr, "[UNINIT] %s\n", msg)
		return
	}

	msg = truncateUTF8(msg, maxLogMessageLength)

	// Merge: fixed fields + context fields + additional fields
	fields := fixedFields
	fields = append(fields, extractFieldsFromContext(ctx)...)
	fields = append(fields, additionalFields...)

	lg.WithOptions(zap.AddCallerSkip(1)).Log(level, msg, fields...)
}
