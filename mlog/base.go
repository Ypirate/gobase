package mlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogConfig struct {
	Level       string `yaml:"level" json:"level"`   // log level: debug/info/warn/error/dpanic/panic/fatal
	Stdout      bool   `yaml:"stdout" json:"stdout"` // whether to output to standard output
	LogDir      string `yaml:"logDir" json:"logDir"`
	LogFileName string `yaml:"logFile" json:"logFile"`
}

var (
	logger      *zap.Logger
	loggerMu    sync.RWMutex
	atomicLevel zap.AtomicLevel
	initialized atomic.Bool
)

// InitLog initializes or re-initializes the global logger.
// It is safe to call multiple times (e.g., in tests or for hot-reload).
func InitLog(conf LogConfig) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	// Flush previous logger before replacing
	if logger != nil {
		_ = logger.Sync()
	}

	atomicLevel = zap.NewAtomicLevelAt(parseLevel(conf.Level))

	// JSON Encoder
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	jsonEncoder := zapcore.NewJSONEncoder(encoderCfg)
	var cores []zapcore.Core

	if conf.Stdout {
		// output stdout
		core := zapcore.NewCore(
			jsonEncoder,
			zapcore.Lock(os.Stdout),
			atomicLevel,
		)
		cores = append(cores, core)
	}
	// when running for the first time, a directory needs to be created  ./logs/app.log
	if conf.LogDir == "" {
		conf.LogDir = "./logs"
	}
	if conf.LogFileName == "" {
		conf.LogFileName = "app.log"
	}
	if err := os.MkdirAll(conf.LogDir, 0755); err != nil {
		// If directory creation fails, fallback to stderr
		fmt.Fprintf(os.Stderr, "failed to create log dir %s: %v\n", conf.LogDir, err)
		core := zapcore.NewCore(
			jsonEncoder,
			zapcore.Lock(os.Stderr),
			atomicLevel,
		)
		cores = append(cores, core)
	} else {
		logFile := filepath.Join(conf.LogDir, conf.LogFileName)
		lumberjackLogger := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    100, // MB
			MaxBackups: 5,
			MaxAge:     7, // days
			Compress:   true,
		}
		core := zapcore.NewCore(
			jsonEncoder,
			zapcore.AddSync(lumberjackLogger),
			atomicLevel,
		)
		cores = append(cores, core)
	}

	// merge cores; enable stacktrace on error+ levels automatically
	core := zapcore.NewTee(cores...)
	logger = zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(2),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	initialized.Store(true)
}

// getLogger returns the global logger in a thread-safe manner.
func getLogger() *zap.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return logger
}

// SetLevel dynamically changes the global log level at runtime.
func SetLevel(level string) {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if initialized.Load() {
		atomicLevel.SetLevel(parseLevel(level))
	}
}

// GetLevel returns the current global log level as a string.
func GetLevel() string {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if initialized.Load() {
		return atomicLevel.Level().String()
	}
	return "uninitialized"
}

// CloseLogger flushes buffered log entries.
func CloseLogger() {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if logger != nil {
		_ = logger.Sync()
	}
}

const maxLogMessageLength = 4096
