// Package telemetry provides structured logging and OpenTelemetry tracing.
package telemetry

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps a zap logger with run-scoped fields and a bundle file sink.
type Logger struct {
	zap  *zap.Logger
	file *os.File
}

// NewLogger creates a Logger that writes JSONL to logs.jsonl in bundleDir.
// Every entry includes the given runId.
func NewLogger(bundleDir, runId string) (*Logger, error) {
	logPath := filepath.Join(bundleDir, "logs.jsonl")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(f),
		zap.DebugLevel,
	)

	z := zap.New(core).With(zap.String("runId", runId))

	return &Logger{zap: z, file: f}, nil
}

// Info logs an info-level message with optional key-value pairs.
func (l *Logger) Info(msg string, kvs ...string) {
	l.zap.Info(msg, toFields(kvs)...)
}

// Warn logs a warn-level message with optional key-value pairs.
func (l *Logger) Warn(msg string, kvs ...string) {
	l.zap.Warn(msg, toFields(kvs)...)
}

// Error logs an error-level message with optional key-value pairs.
func (l *Logger) Error(msg string, kvs ...string) {
	l.zap.Error(msg, toFields(kvs)...)
}

// Close syncs and closes the underlying file.
func (l *Logger) Close() {
	_ = l.zap.Sync()
	l.file.Close()
}

func toFields(kvs []string) []zap.Field {
	fields := make([]zap.Field, 0, len(kvs)/2)
	for i := 0; i+1 < len(kvs); i += 2 {
		fields = append(fields, zap.String(kvs[i], kvs[i+1]))
	}
	return fields
}
