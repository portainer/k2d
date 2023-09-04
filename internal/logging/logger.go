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

// NewLogger creates and configures a new logger.
// It takes the desired log level and a flag that specifies if the logs should be in JSON format.
// The function returns a SugaredLogger and an error if the configuration fails.
func NewLogger(logLevel string, json bool) (*zap.SugaredLogger, error) {
	level, err := parseLogLevel(logLevel)
	if err != nil {
		return nil, err
	}

	config := createLoggerConfig(level, json)

	logger, err := buildLoggerFromConfig(config)
	if err != nil {
		return nil, err
	}

	return setGlobalLogger(logger), nil
}

// parseLogLevel converts a string level to a zapcore.Level type.
func parseLogLevel(logLevel string) (zapcore.Level, error) {
	var level zapcore.Level
	if err := level.Set(logLevel); err != nil {
		return zapcore.InfoLevel, err // Default to InfoLevel
	}
	return level, nil
}

// createLoggerConfig creates a logger configuration based on the given log level and format.
func createLoggerConfig(level zapcore.Level, json bool) zap.Config {
	if json {
		return newJSONLoggerConfig(level)
	}
	return newTextLoggerConfig(level)
}

// newTextLoggerConfig returns a development logger config set at the given log level.
func newTextLoggerConfig(logLevel zapcore.Level) zap.Config {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(logLevel)

	if logLevel != zapcore.DebugLevel {
		config.DisableStacktrace = true
	}

	return config
}

// newJSONLoggerConfig returns a production logger config set at the given log level and with JSON encoding.
func newJSONLoggerConfig(logLevel zapcore.Level) zap.Config {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(logLevel)
	config.Encoding = "json"
	config.DisableStacktrace = true
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	return config
}

// buildLoggerFromConfig builds and returns a logger based on the given configuration.
func buildLoggerFromConfig(config zap.Config) (*zap.Logger, error) {
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// setGlobalLogger sets the logger as the global logger and returns its SugaredLogger.
func setGlobalLogger(logger *zap.Logger) *zap.SugaredLogger {
	zap.ReplaceGlobals(logger)
	return logger.Sugar()
}
