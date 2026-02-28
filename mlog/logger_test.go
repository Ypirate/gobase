package mlog

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	gin.SetMode(gin.TestMode)

	InitLog(LogConfig{
		Level:  "debug",
		Stdout: false,
	})

	w := zapcore.AddSync(&buf)
	origLogger := logger
	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}),
		w,
		zapcore.DebugLevel,
	))
	defer func() { logger = origLogger }()

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

	t.Log("Log levels test passed")
}

func TestContextFields(t *testing.T) {
	var buf bytes.Buffer
	gin.SetMode(gin.TestMode)

	InitLog(LogConfig{
		Level:  "debug",
		Stdout: true,
	})

	w := zapcore.AddSync(&buf)
	origLogger := logger
	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}),
		w,
		zapcore.DebugLevel,
	))
	defer func() { logger = origLogger }()

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

	t.Log("Context fields test passed")
}

func TestGinContextFields(t *testing.T) {
	var buf bytes.Buffer
	gin.SetMode(gin.TestMode)

	InitLog(LogConfig{
		Level:  "debug",
		Stdout: false,
	})

	w := zapcore.AddSync(&buf)
	origLogger := logger
	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}),
		w,
		zapcore.DebugLevel,
	))
	defer func() { logger = origLogger }()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		GinAddString(c, "trace_id", "gin-trace-789")
		Infof(c, "test message with gin context fields")
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)

	output := buf.String()
	if !strings.Contains(output, "gin-trace-789") {
		t.Error("expected trace_id in output")
	}

	t.Log("Gin context fields test passed")
}

func TestNilLoggerFallback(t *testing.T) {
	origLogger := logger
	logger = nil
	defer func() { logger = origLogger }()

	Infof(context.Background(), "this should not panic")

	t.Log("Nil logger fallback test passed")
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

	t.Log("Parse level test passed")
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

	t.Log("Truncate UTF8 test passed")
}
