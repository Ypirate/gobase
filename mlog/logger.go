package mlog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Errorf(ctx any, format string, args ...interface{}) {
	logWithCtx(extractContext(ctx), zapcore.ErrorLevel, format, args...)
}

func Warnf(ctx any, format string, args ...interface{}) {
	logWithCtx(extractContext(ctx), zapcore.WarnLevel, format, args...)
}

func Infof(ctx any, format string, args ...interface{}) {
	logWithCtx(extractContext(ctx), zapcore.InfoLevel, format, args...)
}

func Debugf(ctx any, format string, args ...interface{}) {
	logWithCtx(extractContext(ctx), zapcore.DebugLevel, format, args...)
}

func extractContext(ctx any) context.Context {
	if ctx == nil {
		return nil
	}

	switch v := ctx.(type) {
	case *gin.Context:
		return v.Request.Context()
	case context.Context:
		return v
	default:
		return nil
	}
}

func logWithCtx(ctx context.Context, level zapcore.Level, format string, args ...interface{}) {
	if logger == nil {
		// fallback
		fmt.Fprintf(os.Stderr, "[UNINIT] %s\n", fmt.Sprintf(format, args...))
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = truncateUTF8(msg, maxLogMessageLength)

	var fields = []zap.Field{}
	fields = extractFieldsFromContext(ctx)

	switch level {
	case zapcore.DebugLevel:
		logger.Debug(msg, fields...)
	case zapcore.InfoLevel:
		logger.Info(msg, fields...)
	case zapcore.WarnLevel:
		logger.Warn(msg, fields...)
	case zapcore.ErrorLevel:
		logger.Error(msg, fields...)
	case zapcore.PanicLevel:
		logger.Panic(msg, fields...)
	case zapcore.FatalLevel:
		logger.Fatal(msg, fields...)
	default:
		panic("logger level unhandled default case")
	}
}

func extractFieldsFromContext(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}

	var fields []zap.Field
	fields = getFieldsFromContext(ctx)
	return fields
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
