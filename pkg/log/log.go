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
	logFields := make(logrus.Fields)
	if l.ctx != nil {
		ctxFields := field.FromContext(l.ctx)
		for _, f := range ctxFields {
			logFields[cf.Key()] = cf.Value()
		}
	}
	if len(logFields) > 0 {
		if l.entry != nil {
			l.entry = l.entry.WithFields(logFields)
		} else {
			l.entry = logrus.WithFields(logFields)
		}
	}
	switch l.level {
	case InfoLevel:
		if l.entry != nil {
			l.entry.Info(msg)
		} else {
			logrus.Info(msg)
		}
	case ErrorLevel:
		if l.entry != nil {
			l.entry.Error(msg)
		} else {
			logrus.Error(msg)
		}
	case DebugLevel:
		if l.entry != nil {
			l.entry.Debug(msg)
		} else {
			logrus.Debug(msg)
		}
	}
}

func (l *logger) WithContext(ctx context.Context) Logger {
	l.ctx = ctx
	if l.entry != nil {
		l.entry = l.entry.WithContext(ctx)
	} else {
		l.entry = logrus.WithContext(ctx)
	}
	return l
}

func (l *logger) WithError(err error) Logger {
	l.err = err
	if l.entry != nil {
		l.entry = l.entry.WithError(err)
	} else {
		l.entry = logrus.WithError(err)
	}
	return l
}
