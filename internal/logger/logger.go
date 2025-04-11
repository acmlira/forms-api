package logger

import (
	"go.uber.org/zap"
)

var instance *zap.Logger = func() *zap.Logger {
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	return log
}()

func Fatal(msg string, err error) {
	instance.Fatal(msg, zap.Error(err))
}

func Error(msg string, err error) {
	instance.Error(msg, zap.Error(err))
}

func Info(msg string) {
	instance.Info(msg)
}
