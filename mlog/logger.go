package mlog

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Errorf(ctx context.Context, format string, args ...interface{}) {
	logWithCtx(ctx, zapcore.ErrorLevel, format, args...)
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

func logWithCtx(ctx context.Context, level zapcore.Level, format string, args ...interface{}) {
	if logger == nil {
		// fallback
		fmt.Fprintf(os.Stderr, "[UNINIT] %s\n", fmt.Sprintf(format, args...))
		return
	}

	var fields = []zap.Field{}
	fields = extractFieldsFromContext(ctx)

	msg := fmt.Sprintf(format, args...)

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

	// example: trace_id
	if v := ctx.Value("trace_id"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			fields = append(fields, zap.String("trace_id", s))
		}
	}

	// example: request_id
	if v := ctx.Value("request_id"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			fields = append(fields, zap.String("request_id", s))
		}
	}

	// example: user_id
	if v := ctx.Value("user_id"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			fields = append(fields, zap.String("user_id", s))
		}
	}

	// TODO: extend other fields，example: tenant_id, app_name
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
