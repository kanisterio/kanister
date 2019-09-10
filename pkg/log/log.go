package log

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// Field type TBD.
type F interface{}

type logLevel string

const (
	logTypeInfo        logLevel = "Info"
	logTypeError       logLevel = "Error"
	logTypeDebug       logLevel = "Debug"
	logTypeWithFields  logLevel = "WithFields"
	logTypeWithContext logLevel = "WithContext"
	logTypeWithError   logLevel = "WithError"
)

type logger struct {
	level  logLevel
	entry  *logrus.Entry
	fields []F
}

func Info() Printer {
	return logger{level: logTypeInfo}
}

func Error() Printer {
	return logger{level: logTypeError}
}

func Debug() Printer {
	return logger{level: logTypeDebug}
}

func WithFields(fields ...F) Printer {
	return logger{
		level:  logTypeWithFields,
		fields: fields,
	}
}

func WithContext(ctx) Printer {
	return logger{
		level: logTypeWithContext,
		entry: logrus.WithContext(ctx),
	}
}

func WithError(err Error) Printer {
	return logger{
		level: logTypeWithError,
		entry: logrus.WithError(err),
	}
}

// Most commonly used logging function
func Print(msg string, fields ...F) {
	Info().Print(msg, fields...)
}

func (l logger) Print(msg string, fields ...F) {
	//message := msg
	var ctx context
	for _, field := range fields {

		// add code to  Process fields
	}
	//TODO add fields to the logs
	switch l.level {
	case logTypeInfo:
		logrus.Info(msg...)
	case logTypeError:
		logrus.Error(msg...)
	case logTypeDebug:
		logrus.Debug(msg...)
	case logTypeWithFields:
		if len(l.fields) != 0 {
			Print(msg, fields...)
		}
	case logTypeWithContext:
		l.entry.Info(msg...)
	case logTypeWithError:
		l.entry.Error(fmt.Errorf(msg))
	}
}
