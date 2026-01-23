package mlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

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
	logger *zap.Logger
	once   sync.Once
)

// InitLog Init log
func InitLog(conf LogConfig) {
	once.Do(func() {
		level := parseLevel(conf.Level)

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
			EncodeTime:     zapcore.RFC3339TimeEncoder,
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
				level,
			)
			cores = append(cores, core)
		} else {
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
					level,
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
					level,
				)
				cores = append(cores, core)
			}
		}

		// merge cores
		core := zapcore.NewTee(cores...)
		logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))
	})
}

// CloseLogger Refresh and close logs
func CloseLogger() {
	if logger != nil {
		_ = logger.Sync()
	}
}
