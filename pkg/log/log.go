package log

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/kanisterio/kanister/pkg/caller"
	"github.com/kanisterio/kanister/pkg/config"
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

	// LevelVarName is the environment variable that controls
	// init log levels
	LevelEnvName = "LOG_LEVEL"
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
		"SERVICE_NAME",
		"VERSION",
	}
	for _, e := range envVars {
		if ev, ok := os.LookupEnv(e); ok {
			envVarFields = field.Add(envVarFields, strings.ToLower(e), ev)
		}
	}
}

// SetupClusterNameInLogVars sets up the `cluster_name` field in `envVarFields`
// so that it can be printed with the logs.
func SetupClusterNameInLogVars() {
	if clsName, err := config.GetClusterName(nil); err == nil {
		envVarFields = field.Add(envVarFields, "cluster_name", clsName)
	}
}

// OutputFormat sets the output data format.
type OutputFormat uint8

const (
	// TextFormat creates a plain text format log entry (not CEE).
	TextFormat OutputFormat = iota
	// JSONFormat create a JSON format log entry.
	JSONFormat
)

// SetFormatter sets the output formatter.
func SetFormatter(format OutputFormat) {
	switch format {
	case TextFormat:
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano})
	case JSONFormat:
		log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	default:
		panic("not implemented")
	}
}

func init() {
	SetFormatter(JSONFormat)
	initEnvVarFields()
	initLogLevel()
}

func initLogLevel() {
	level, err := logrus.ParseLevel(os.Getenv(LevelEnvName))
	if err != nil {
		level = logrus.InfoLevel
	}
	SetLevel(Level(level))
}

// SetLevel sets the current log level.
func SetLevel(level Level) {
	log.SetLevel(logrus.Level(level))
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

// PrintTo works just like Print. It allows caller to specify the writer to use
// to output the log.
func PrintTo(w io.Writer, msg string, fields ...field.M) {
	Info().PrintTo(w, msg, fields...)
}

func WithContext(ctx context.Context) Logger {
	return Info().WithContext(ctx)
}

func WithError(err error) Logger {
	return Info().WithError(err)
}

func (l *logger) Print(msg string, fields ...field.M) {
	entry := l.entry(fields...)
	entry.Logln(logrus.Level(l.level), msg)
}

func (l *logger) PrintTo(w io.Writer, msg string, fields ...field.M) {
	entry := l.entry(fields...)
	entry.Logger.SetOutput(w)
	entry.Logln(logrus.Level(l.level), msg)
}

func (l *logger) entry(fields ...field.M) *logrus.Entry {
	logFields := make(logrus.Fields)
	if envVarFields != nil {
		for _, f := range envVarFields.Fields() {
			logFields[f.Key()] = f.Value()
		}
	}

	frame := caller.GetFrame(4)
	logFields["Function"] = frame.Function
	logFields["File"] = frame.File
	logFields["Line"] = frame.Line

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

	// use a cloned logger for this entry, so that any changes to this clone
	// (e.g., via SetOutput()) will not affect the global logger.
	cloned := cloneGlobalLogger()
	entry := cloned.WithFields(logFields)
	if l.err != nil {
		entry = entry.WithError(l.err)
	}
	return entry
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

func cloneGlobalLogger() *logrus.Logger {
	cloned := logrus.New()
	cloned.SetFormatter(log.Formatter)
	cloned.SetReportCaller(log.ReportCaller)
	cloned.SetLevel(log.Level)
	cloned.SetOutput(log.Out)
	cloned.ExitFunc = log.ExitFunc

	globalHooks := make(map[logrus.Hook]bool)

	for _, hooks := range log.Hooks {
		for _, hook := range hooks {
			globalHooks[hook] = true
		}
	}

	for hook := range globalHooks {
		cloned.Hooks.Add(hook)
	}

	return cloned
}
