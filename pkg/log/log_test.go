package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/field"
)

const (
	infoLevelStr = "info"
)

type LogSuite struct{}

var _ = Suite(&LogSuite{})

func Test(t *testing.T) {
	TestingT(t)
}

func (s *LogSuite) TestWithNilError(c *C) {
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	// Should not panic
	WithError(nil).Print("Message")
}

func (s *LogSuite) TestWithNilContext(c *C) {
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	// Should not panic
	WithContext(nil).Print("Message") //nolint:staticcheck
}

func (s *LogSuite) TestLogMessage(c *C) {
	const text = "Some useful text."
	testLogMessage(c, text, Print)
}

func (s *LogSuite) TestLogWithFields(c *C) {
	const text = "Some useful text."
	entry := testLogMessage(c, text, Print, field.M{"key": "value"})
	c.Assert(entry["level"], Equals, infoLevelStr)
	// Error should not be set in the log entry
	c.Assert(entry["error"], Equals, nil)
	// A field with "key" should be set in the log entry
	c.Assert(entry["key"], Equals, "value")
}

func (s *LogSuite) TestLogWithError(c *C) {
	const text = "My error message"
	err := errors.New("test error")
	entry := testLogMessage(c, text, WithError(err).Print)
	c.Assert(entry["error"], Equals, err.Error())
	c.Assert(entry["level"], Equals, infoLevelStr)
}

func (s *LogSuite) TestLogWithContext(c *C) {
	const text = "My error message"
	ctx := context.Background()
	entry := testLogMessage(c, text, WithContext(ctx).Print)
	c.Assert(entry["level"], Equals, infoLevelStr)
	// Error should not be set in the log entry
	c.Assert(entry["error"], Equals, nil)
}

func (s *LogSuite) TestLogWithContextFields(c *C) {
	const text = "My error message"
	ctx := field.Context(context.Background(), "key", "value")
	entry := testLogMessage(c, text, WithContext(ctx).Print)
	c.Assert(entry["level"], Equals, infoLevelStr)
	// Error should not be set in the log entry
	c.Assert(entry["error"], Equals, nil)
	// A field with "key" should be set in the log entry
	c.Assert(entry["key"], Equals, "value")
}

func (s *LogSuite) TestLogWithContextFieldsAndError(c *C) {
	const text = "My error message"
	ctx := field.Context(context.Background(), "key", "value")
	err := errors.New("test error")
	entry := testLogMessage(c, text, WithError(err).WithContext(ctx).Print)
	c.Assert(entry["level"], Equals, infoLevelStr)
	// Error should be included in the log entry
	c.Assert(entry["error"], Equals, err.Error())
	// A field with "key" should be set in the log entry
	c.Assert(entry["key"], Equals, "value")
}

func (s *LogSuite) TestLogPrintTo(c *C) {
	buf := &bytes.Buffer{}
	msg := "test log message"
	fields := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
	}
	PrintTo(buf, msg, fields)

	entry := map[string]interface{}{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	c.Assert(err, IsNil)
	c.Assert(entry, NotNil)
	c.Assert(entry["msg"], Equals, msg)
}

func (s *LogSuite) TestLogPrintToParallel(c *C) {
	// this test ensures that the io.Writer passed to PrintTo() doesn't override
	// that of the global logger.
	// previously, the entry() function would return an entry bound to the global
	// logger where changes made to the entry's logger yields a global effect.
	// see https://github.com/kanisterio/kanister/issues/1523.

	var (
		msg     = "test log message"
		buffers = []*bytes.Buffer{
			{},
			{},
		}
		wg = sync.WaitGroup{}
	)

	for i := 0; i < len(buffers); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			fields := map[string]interface{}{
				"field1": "value1",
				"field2": "value2",
			}
			PrintTo(buffers[i], fmt.Sprintf("%s %d", msg, i), fields)
		}(i)
	}
	wg.Wait()

	for i := 0; i < len(buffers); i++ {
		actual := map[string]interface{}{}
		err := json.Unmarshal(buffers[i].Bytes(), &actual)
		c.Assert(err, IsNil)
		c.Assert(actual, NotNil)
		c.Assert(actual["msg"], Equals, fmt.Sprintf("%s %d", msg, i))
	}
}

func testLogMessage(c *C, msg string, print func(string, ...field.M), fields ...field.M) map[string]interface{} {
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	var memLog bytes.Buffer
	log.SetOutput(&memLog)
	print(msg, fields...)
	var entry map[string]interface{}
	err := json.Unmarshal(memLog.Bytes(), &entry)
	c.Assert(err, IsNil)
	c.Assert(entry, NotNil)
	c.Assert(entry["msg"], Equals, msg)
	return entry
}

func (s *LogSuite) TestLogLevel(c *C) {
	err := os.Unsetenv(LevelEnvName)
	c.Assert(err, IsNil)
	initLogLevel()
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})

	var output bytes.Buffer
	log.SetOutput(&output)
	ctx := field.Context(context.Background(), "key", "value")
	var entry map[string]interface{}
	// Check if debug level log is printed when log level is info
	Debug().WithContext(ctx).Print("Testing debug level")
	err = json.Unmarshal(output.Bytes(), &entry)

	c.Assert(err, NotNil)
	c.Assert(output.String(), HasLen, 0)

	// Check if debug level log is printed when log level is debug
	err = os.Setenv(LevelEnvName, "debug")
	c.Assert(err, IsNil)
	defer func() {
		err := os.Unsetenv(LevelEnvName)
		c.Assert(err, IsNil)
		initLogLevel()
	}()
	initLogLevel()
	Debug().WithContext(ctx).Print("Testing debug level")

	cerr := json.Unmarshal(output.Bytes(), &entry)
	c.Assert(cerr, IsNil)
	c.Assert(entry, NotNil)
	c.Assert(entry["msg"], Equals, "Testing debug level")
}

func (s *LogSuite) TestCloneGlobalLogger(c *C) {
	hook := newTestLogHook()
	log.AddHook(hook)
	actual := cloneGlobalLogger()
	c.Assert(actual.Formatter, Equals, log.Formatter)
	c.Assert(actual.ReportCaller, Equals, log.ReportCaller)
	c.Assert(actual.Level, Equals, log.Level)
	c.Assert(actual.Out, Equals, log.Out)
	c.Assert(actual.Hooks, DeepEquals, log.Hooks)

	// changing `actual` should not affect global logger
	actual.SetFormatter(&logrus.TextFormatter{})
	actual.SetReportCaller(true)
	actual.SetLevel(logrus.ErrorLevel)
	actual.SetOutput(&bytes.Buffer{})
	actual.AddHook(&logHook{})

	c.Assert(actual.Formatter, Not(Equals), log.Formatter)
	c.Assert(actual.ReportCaller, Not(Equals), log.ReportCaller)
	c.Assert(actual.Level, Not(Equals), log.Level)
	c.Assert(actual.Out, Not(Equals), log.Out)
	c.Assert(actual.Hooks, Not(DeepEquals), log.Hooks)

	log.Println("Test message")
	c.Assert(len(hook.capturedMessages), Equals, 1)
	c.Assert(hook.capturedMessages[0].Message, Equals, "Test message")
}

type logHook struct {
	capturedMessages []*logrus.Entry
}

func newTestLogHook() *logHook {
	return &logHook{
		capturedMessages: make([]*logrus.Entry, 0),
	}
}

func (t *logHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

func (t *logHook) Fire(entry *logrus.Entry) error {
	if t.capturedMessages != nil {
		t.capturedMessages = append(t.capturedMessages, entry)
	}
	return nil
}
