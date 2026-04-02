package mlog

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogLevels(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	ctx := context.Background()
	Debugf(ctx, "debug message %s", "test")
	Infof(ctx, "info message %s", "test")
	Warnf(ctx, "warn message %s", "test")
	Errorf(ctx, "error message %s", "test")

	output := buf.String()
	if !strings.Contains(output, `"level":"debug"`) {
		t.Error("expected debug level in output")
	}
	if !strings.Contains(output, `"level":"info"`) {
		t.Error("expected info level in output")
	}
	if !strings.Contains(output, `"level":"warn"`) {
		t.Error("expected warn level in output")
	}
	if !strings.Contains(output, `"level":"error"`) {
		t.Error("expected error level in output")
	}
}

func TestContextFields(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	ctx := context.Background()
	ctx = AddFields(ctx, zap.String("trace_id", "abc123"), zap.String("user_id", "user456"))
	Infof(ctx, "test message with context fields")

	output := buf.String()
	if !strings.Contains(output, "abc123") {
		t.Error("expected trace_id in output")
	}
	if !strings.Contains(output, "user456") {
		t.Error("expected user_id in output")
	}
}

func TestGinContextFields(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		ctx := AddFields(c.Request.Context(), zap.String("trace_id", "gin-trace-789"))
		c.Request = c.Request.WithContext(ctx)
		Infof(c.Request.Context(), "test message with gin context fields")
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)

	output := buf.String()
	if !strings.Contains(output, "gin-trace-789") {
		t.Error("expected trace_id in output")
	}
}

func TestNilLoggerFallback(t *testing.T) {
	loggerMu.Lock()
	origLogger := logger
	logger = nil
	loggerMu.Unlock()

	defer func() {
		loggerMu.Lock()
		logger = origLogger
		loggerMu.Unlock()
	}()

	Infof(context.Background(), "this should not panic")
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"INFO", zapcore.InfoLevel},
		{"WARN", zapcore.WarnLevel},
		{"warning", zapcore.WarnLevel},
		{"ERROR", zapcore.ErrorLevel},
		{"dpanic", zapcore.DPanicLevel},
		{"panic", zapcore.PanicLevel},
		{"fatal", zapcore.FatalLevel},
		{"unknown", zapcore.InfoLevel},
		{"", zapcore.InfoLevel},
	}

	for _, tt := range tests {
		result := parseLevel(tt.input)
		if result != tt.expected {
			t.Errorf("parseLevel(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestTruncateUTF8(t *testing.T) {
	short := "hello"
	if truncateUTF8(short, 10) != short {
		t.Error("truncate should not affect short string")
	}

	long := strings.Repeat("a", 5000)
	truncated := truncateUTF8(long, 100)
	if len(truncated) > 150 {
		t.Error("truncated string too long")
	}
	if !strings.Contains(truncated, "truncated") {
		t.Error("truncated string should contain truncation marker")
	}
}

func TestFatalfWithContext(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	// Replace logger with one that uses WriteThenPanic hook
	loggerMu.Lock()
	w := zapcore.AddSync(buf)
	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(testEncoderConfig()),
		w,
		zapcore.DebugLevel,
	), zap.WithFatalHook(zapcore.WriteThenPanic))
	loggerMu.Unlock()

	ctx := context.Background()
	ctx = AddFields(ctx, zap.String("trace_id", "fatal-trace-123"))

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Fatalf triggered panic as expected: %v", r)
		}
	}()

	Fatalf(ctx, "critical failure: database connection lost")

	output := buf.String()
	if !strings.Contains(output, `"level":"fatal"`) {
		t.Error("expected fatal level in output")
	}
	if !strings.Contains(output, "fatal-trace-123") {
		t.Error("expected trace_id in output")
	}
}

func TestDynamicLogLevel(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	// Initially at debug level
	Debugf(context.Background(), "debug message 1")
	if !strings.Contains(buf.String(), "debug message 1") {
		t.Error("expected debug message at debug level")
	}

	// Change to error level - need to recreate logger with new level
	buf.Reset()
	loggerMu.Lock()
	w := zapcore.AddSync(buf)
	atomicLevel = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(testEncoderConfig()),
		w,
		atomicLevel,
	))
	loggerMu.Unlock()

	Debugf(context.Background(), "debug message 2")
	Infof(context.Background(), "info message 2")
	Errorf(context.Background(), "error message 2")

	output := buf.String()
	if strings.Contains(output, "debug message 2") {
		t.Error("should not log debug at error level")
	}
	if strings.Contains(output, "info message 2") {
		t.Error("should not log info at error level")
	}
	if !strings.Contains(output, "error message 2") {
		t.Error("should log error at error level")
	}

	// Verify GetLevel
	if GetLevel() != "error" {
		t.Errorf("expected level 'error', got '%s'", GetLevel())
	}
}

func TestChildLogger(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	childLogger := With(
		zap.String("module", "user"),
		zap.String("service", "auth"),
	)

	ctx := context.Background()
	ctx = AddFields(ctx, zap.String("trace_id", "child-trace-456"))

	childLogger.Infof(ctx, "user login attempt")

	output := buf.String()
	if !strings.Contains(output, "user login attempt") {
		t.Error("expected log message")
	}
	if !strings.Contains(output, `"module":"user"`) {
		t.Error("expected module field")
	}
	if !strings.Contains(output, `"service":"auth"`) {
		t.Error("expected service field")
	}
	if !strings.Contains(output, "child-trace-456") {
		t.Error("expected trace_id from context")
	}
}

func TestChildLoggerAllLevels(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	childLogger := With(zap.String("component", "test"))
	ctx := context.Background()

	childLogger.Debugf(ctx, "debug msg")
	childLogger.Infof(ctx, "info msg")
	childLogger.Warnf(ctx, "warn msg")
	childLogger.Errorf(ctx, "error msg")

	output := buf.String()
	if !strings.Contains(output, "debug msg") {
		t.Error("expected debug message")
	}
	if !strings.Contains(output, "info msg") {
		t.Error("expected info message")
	}
	if !strings.Contains(output, "warn msg") {
		t.Error("expected warn message")
	}
	if !strings.Contains(output, "error msg") {
		t.Error("expected error message")
	}
}
