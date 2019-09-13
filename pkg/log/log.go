package log

import (
	"context"

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
	ctx   context.Context
	err   error
}

func Info() Logger {
	return &logger{level: InfoLevel}
}

func Error() Logger {
	return &logger{level: ErrorLevel}
}

func Debug() Logger {
	return &logger{level: DebugLevel}
}

// Print adds `msg` to the log at `InfoLevel`. It is a wrapper for `Info().Print(msg)`, since this is the most common use case.
func Print(msg string) {
	Info().Print(msg)
}

func WithContext(ctx context.Context) {
	Info().WithContext(ctx)
}

func (l *logger) Print(msg string) {
	switch l.level {
	case InfoLevel:
		logrus.Info(msg)
	case ErrorLevel:
		logrus.Error(msg)
	case DebugLevel:
		logrus.Debug(msg)
	}
}

func (l *logger) WithContext(ctx context.Context) Logger {
	l.ctx = ctx
	return l
}

func (l *logger) WithError(err error) Logger {
	l.err = err
	return l
}
