package log

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/pkg/field"
)

type logLevel string

const (
	logTypeInfo  logLevel = "Info"
	logTypeError logLevel = "Error"
	logTypeDebug logLevel = "Debug"
)

type logger struct {
	level logLevel
	entry *logrus.Entry
	ctx   context
	err   error
}

func Info() Printer {
	return &logger{level: logTypeInfo}
}

func Error() Printer {
	return &logger{level: logTypeError}
}

func Debug() Printer {
	return &logger{level: logTypeDebug}
}

// Most commonly used logging function
func Print(msg string) {
	Info().Print(msg)
}

func WithContext(ctx) {
	Info().WithContext(ctx)
}

func (l *logger) Print(msg string) {
	switch l.level {
	case logTypeInfo:
		logrus.Info(msg...)
	case logTypeError:
		logrus.Error(msg...)
	case logTypeDebug:
		logrus.Debug(msg...)
	}
}

func (l *logger) WithContext(ctx context) Printer {
	l.ctx = ctx
	return l
}

func (l *logger) WithError(err Error) Printer {
	l.err = err
	return l
}
