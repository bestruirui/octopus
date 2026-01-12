package log

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger
var atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)

// levelFirstEncoder is a custom encoder that outputs [LEVEL] first,
// followed by the timestamp in RFC3339-like format.
func levelFirstEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
	enc.AppendString(time.Now().Format("2006-01-02T15:04:05"))
}

// TimeKey is empty to disable zap's default time output at the beginning.
// EncodeTime is disabled since TimeKey is empty.
var consoleEncoder = zapcore.EncoderConfig{
	TimeKey:       "",
	LevelKey:      "level",
	MessageKey:    "msg",
	CallerKey:     "caller",
	StacktraceKey: "stacktrace",
	EncodeLevel:   levelFirstEncoder,
	// EncodeTime:    zapcore.RFC3339TimeEncoder,
	EncodeCaller: zapcore.ShortCallerEncoder,
}

func init() {
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleEncoder),
		zapcore.AddSync(os.Stdout),
		atomicLevel,
	)
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zap.ErrorLevel),
	}
	Logger = zap.New(core, opts...).Sugar()
}

func SetLevel(level string) {
	var lvl zapcore.Level
	err := lvl.UnmarshalText([]byte(level))
	if err != nil {
		return
	}
	atomicLevel.SetLevel(lvl)
}

func Infof(template string, args ...interface{}) {
	Logger.Infof(template, args...)
}

func Errorf(template string, args ...interface{}) {
	Logger.Errorf(template, args...)
}

func Warnf(template string, args ...interface{}) {
	Logger.Warnf(template, args...)
}

func Debugf(template string, args ...interface{}) {
	Logger.Debugf(template, args...)
}
