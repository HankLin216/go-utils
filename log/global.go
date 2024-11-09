package log

import (
	"fmt"

	"go.uber.org/zap"
)

var globalLog *zap.Logger

func init() {
	globalLog, _ = zap.NewProduction()
}

func SetLogger(l *zap.Logger) {
	if l == nil {
		panic(fmt.Errorf("nil logger"))
	}

	globalLog = l
}

func Debug(msg string, fields ...zap.Field) {
	globalLog.Debug(msg, fields...)
}

func Debugf(template string, args ...interface{}) {
	globalLog.Sugar().Debugf(template, args...)
}

func Info(msg string, fields ...zap.Field) {
	globalLog.Info(msg, fields...)
}

func Infof(template string, args ...interface{}) {
	globalLog.Sugar().Infof(template, args...)
}

func Warn(msg string, fields ...zap.Field) {
	globalLog.Warn(msg, fields...)
}

func Warnf(template string, args ...interface{}) {
	globalLog.Sugar().Warnf(template, args...)
}

func Error(msg string, fields ...zap.Field) {
	globalLog.Error(msg, fields...)
}

func Errorf(template string, args ...interface{}) {
	globalLog.Sugar().Errorf(template, args...)
}

func Fatal(msg string, fields ...zap.Field) {
	globalLog.Fatal(msg, fields...)
}

func Fatalf(template string, args ...interface{}) {
	globalLog.Sugar().Fatalf(template, args...)
}
