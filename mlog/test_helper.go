package mlog

import (
	"bytes"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// setupTestLogger creates a test logger that writes to a buffer.
// Returns the buffer and a cleanup function.
func setupTestLogger(t *testing.T) (*bytes.Buffer, func()) {
	var buf bytes.Buffer
	gin.SetMode(gin.TestMode)

	InitLog(LogConfig{Level: "debug", Stdout: true})

	w := zapcore.AddSync(&buf)
	origLogger := getLogger()

	testLogger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(testEncoderConfig()),
		w,
		zapcore.DebugLevel,
	))

	loggerMu.Lock()
	logger = testLogger
	loggerMu.Unlock()

	cleanup := func() {
		loggerMu.Lock()
		logger = origLogger
		loggerMu.Unlock()
	}

	return &buf, cleanup
}

// testEncoderConfig returns a standard encoder config for tests.
func testEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}
