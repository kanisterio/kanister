package log

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/pkg/errors"
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

// OutputSink describes the current output sink.
type OutputSink uint8

// Valid log sinks: stderr or fluentbit
const (
	StderrSink OutputSink = iota
	FluentbitSink
)

// Names of environment variables to configure the logging sink
const (
	LoggingServiceHostEnv = "LOGGING_SVC_SERVICE_HOST"
	LoggingServicePortEnv = "LOGGING_SVC_SERVICE_PORT_LOGGING"
)

type logger struct {
	level Level
	ctx   context.Context
	err   error
}

// common logger implementation used in the library
var log = logrus.New()

// SetOutput sets the output destination.
func SetOutput(sink OutputSink) error {
	switch sink {
	case StderrSink:
		log.SetOutput(os.Stderr)
		return nil
	case FluentbitSink:
		fbitAddr, ok := os.LookupEnv(LoggingServiceHostEnv)
		if !ok {
			return errors.New("Unable to find Fluentbit host address")
		}
		fbitPort, ok := os.LookupEnv(LoggingServicePortEnv)
		if !ok {
			return errors.New("Unable to find Fluentbit logging port")
		}
		hook := NewFluentbitHook(fbitAddr + ":" + fbitPort)
		log.AddHook(hook)
		return nil
	default:
		return errors.New("not implemented")
	}
}

var envVarFields field.Fields

// initEnvVarFields populates envVarFields with values from the host's environment.
func initEnvVarFields() {
	envVars := []string{
		"HOSTNAME",
		"CLUSTER_NAME",
		"SERVICE_NAME",
		"VERSION",
	}
	for _, e := range envVars {
		if ev, ok := os.LookupEnv(e); ok {
			envVarFields = field.Add(envVarFields, strings.ToLower(e), ev)
		}
	}
}

func init() {
	initEnvVarFields()
}

func Info() Logger {
	return &logger{
		level: InfoLevel,
	}
}

func Error() Logger {
	return &logger{
		level: ErrorLevel,
	}
}

func Debug() Logger {
	return &logger{
		level: DebugLevel,
	}
}

// Print adds `msg` to the log at `InfoLevel`. It is a wrapper for `Info().Print(msg)`, since this is the most common use case.
func Print(msg string, fields ...field.M) {
	Info().Print(msg, fields...)
}

func WithContext(ctx context.Context) Logger {
	return Info().WithContext(ctx)
}

func WithError(err error) Logger {
	return Info().WithError(err)
}

func (l *logger) Print(msg string, fields ...field.M) {
	logFields := make(logrus.Fields)

	for _, f := range envVarFields.Fields() {
		logFields[f.Key()] = f.Value()
	}

	if ctxFields := field.FromContext(l.ctx); ctxFields != nil {
		for _, cf := range ctxFields.Fields() {
			logFields[cf.Key()] = cf.Value()
		}
	}

	for _, f := range fields {
		for k, v := range f {
			logFields[k] = v
		}
	}

	entry := log.WithFields(logFields)
	if l.err != nil {
		entry = entry.WithError(l.err)
	}
	entry.Logln(logrus.Level(l.level), msg)
}

func (l *logger) WithContext(ctx context.Context) Logger {
	l.ctx = ctx
	return l
}

func (l *logger) WithError(err error) Logger {
	l.err = err
	return l
}

// Scrapes fields of interest from the logrus.Entry and converts then into a JSON []byte.
func entryToJSON(entry *logrus.Entry) []byte {
	data := make(logrus.Fields, len(entry.Data)+3)

	data["Message"] = entry.Message
	data["Level"] = entry.Level.String()
	data["Time"] = entry.Time

	for k, v := range entry.Data {
		data[k] = v
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	n := []byte("\n")
	bytes = append(bytes, n...)

	return bytes
}
