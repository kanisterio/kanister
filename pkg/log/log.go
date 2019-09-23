package log

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kanisterio/kanister/pkg/field"
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

// common logger implementation used in the library
var log = logrus.New()

func Info() Logger {
	return &logger{
		level: InfoLevel,
		entry: logrus.NewEntry(logrus.New()),
	}
}

func Error() Logger {
	return &logger{
		level: ErrorLevel,
		entry: logrus.NewEntry(logrus.New()),
	}
}

func Debug() Logger {
	return &logger{
		level: DebugLevel,
		entry: logrus.NewEntry(logrus.New()),
	}
}

// Print adds `msg` to the log at `InfoLevel`. It is a wrapper for `Info().Print(msg)`, since this is the most common use case.
func Print(msg string) {
	Info().Print(msg)
}

func WithContext(ctx context.Context) {
	Info().WithContext(ctx)
}

func WithError(err error) {
	Info().WithError(err)
}

func (l *logger) Print(msg string) {
	logFields := make(logrus.Fields)
	if ctxFields := field.FromContext(l.ctx); ctxFields != nil {
		for _, cf := range ctxFields.Fields() {
			logFields[cf.Key()] = cf.Value()
		}
	}
	entry := log.WithFields(logFields)

	switch l.level {
	case InfoLevel:
		entry.Info(msg)
	case ErrorLevel:
		entry.Error(msg)
	case DebugLevel:
		entry.Debug(msg)
	}
}

func (l *logger) WithContext(ctx context.Context) Logger {
	l.ctx = ctx
	l.entry = l.entry.WithContext(ctx)
	return l
}

func (l *logger) WithError(err error) Logger {
	l.err = err
	l.entry = l.entry.WithError(err)
	return l
}
