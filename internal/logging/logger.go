package logging

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxLogger struct{}

// ContextWithLogger adds logger to context
func ContextWithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, ctxLogger{}, logger)
}

// LoggerFromContext returns logger from context
func LoggerFromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(ctxLogger{}).(*zap.SugaredLogger); ok {
		return logger
	}

	// Default logger implementation
	return zap.S()
}

// NewLogger2 creates a new instance of a logger with the specified log level and format.
// The function takes two parameters: logLevel (string) and json (bool).
// logLevel represents the desired log level for the logger.
// json is a flag indicating whether the logger should output logs in JSON format.
// The function returns a pointer to a zap.SugaredLogger and an error if any.
// If successful, the returned logger is set as the global logger using zap.ReplaceGlobals.
func NewLogger(logLevel string, json bool) (*zap.SugaredLogger, error) {
	var level = zapcore.InfoLevel
	err := level.Set(logLevel)
	if err != nil {
		return nil, err
	}

	// TODO: remove stacktrace from INFO level logs

	var config zap.Config
	if json {
		config = newJSONLoggerConfig(level)
	} else {
		config = newTextLoggerConfig(level)
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	zap.ReplaceGlobals(logger)
	return logger.Sugar(), nil
}

func newTextLoggerConfig(logLevel zapcore.Level) zap.Config {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(logLevel)
	return config
}

func newJSONLoggerConfig(logLevel zapcore.Level) zap.Config {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(logLevel)
	config.Encoding = "json"
	config.DisableStacktrace = true
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.DateTime)
	return config
}
