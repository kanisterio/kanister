package log

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// Level describes the current log level.
type Level uint32

const (
	// DebugLevel log level.
	DebugLevel Level = Level(logrus.DebugLevel)
	// InfoLevel log level.
	InfoLevel Level = Level(logrus.InfoLevel)
	// ErrorLevel log level.
	ErrorLevel Level = Level(logrus.ErrorLevel)
)

type logger struct {
	level Level
	entry *logrus.Entry
	ctx   context
	err   error
}

func Info() Printer {
	return &logger{level: InfoLevel}
}

func Error() Printer {
	return &logger{level: ErrorLevel}
}

func Debug() Printer {
	return &logger{level: DebugLevel}
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
	case InfoLevel:
		logrus.Info(msg...)
	case logTypeError:
		ErrorLevel.Error(msg...)
	case DebugLevel:
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
