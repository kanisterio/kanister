package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/kanisterio/errkit"
	"github.com/sirupsen/logrus"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/field"
)

const (
	infoLevelStr = "info"
)

type LogSuite struct{}

var _ = check.Suite(&LogSuite{})

func Test(t *testing.T) {
	check.TestingT(t)
}

func (s *LogSuite) TestWithNilError(c *check.C) {
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	// Should not panic
	WithError(nil).Print("Message")
}

func (s *LogSuite) TestWithNilContext(c *check.C) {
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	// Should not panic
	WithContext(nil).Print("Message") //nolint:staticcheck
}

func (s *LogSuite) TestLogMessage(c *check.C) {
	const text = "Some useful text."
	testLogMessage(c, text, Print)
}

func (s *LogSuite) TestLogWithFields(c *check.C) {
	const text = "Some useful text."
	entry := testLogMessage(c, text, Print, field.M{"key": "value"})
	c.Assert(entry["level"], check.Equals, infoLevelStr)
	// Error should not be set in the log entry
	c.Assert(entry["error"], check.Equals, nil)
	// A field with "key" should be set in the log entry
	c.Assert(entry["key"], check.Equals, "value")
}

func (s *LogSuite) TestLogWithError(c *check.C) {
	const text = "My error message"
	err := errkit.New("test error")
	entry := testLogMessage(c, text, WithError(err).Print)
	c.Assert(entry["error"], check.Equals, err.Error())
	c.Assert(entry["level"], check.Equals, infoLevelStr)
}

func (s *LogSuite) TestLogWithContext(c *check.C) {
	const text = "My error message"
	ctx := context.Background()
	entry := testLogMessage(c, text, WithContext(ctx).Print)
	c.Assert(entry["level"], check.Equals, infoLevelStr)
	// Error should not be set in the log entry
	c.Assert(entry["error"], check.Equals, nil)
}

func (s *LogSuite) TestLogWithContextFields(c *check.C) {
	const text = "My error message"
	ctx := field.Context(context.Background(), "key", "value")
	entry := testLogMessage(c, text, WithContext(ctx).Print)
	c.Assert(entry["level"], check.Equals, infoLevelStr)
	// Error should not be set in the log entry
	c.Assert(entry["error"], check.Equals, nil)
	// A field with "key" should be set in the log entry
	c.Assert(entry["key"], check.Equals, "value")
}

func (s *LogSuite) TestLogWithContextFieldsAndError(c *check.C) {
	const text = "My error message"
	ctx := field.Context(context.Background(), "key", "value")
	err := errkit.New("test error")
	entry := testLogMessage(c, text, WithError(err).WithContext(ctx).Print)
	c.Assert(entry["level"], check.Equals, infoLevelStr)
	// Error should be included in the log entry
	c.Assert(entry["error"], check.Equals, err.Error())
	// A field with "key" should be set in the log entry
	c.Assert(entry["key"], check.Equals, "value")
}

func (s *LogSuite) TestLogPrintTo(c *check.C) {
	buf := &bytes.Buffer{}
	msg := "test log message"
	fields := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
	}
	PrintTo(buf, msg, fields)

	entry := map[string]interface{}{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	c.Assert(err, check.IsNil)
	c.Assert(entry, check.NotNil)
	c.Assert(entry["msg"], check.Equals, msg)
}

func (s *LogSuite) TestLogPrintToParallel(c *check.C) {
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
		c.Assert(err, check.IsNil)
		c.Assert(actual, check.NotNil)
		c.Assert(actual["msg"], check.Equals, fmt.Sprintf("%s %d", msg, i))
	}
}

func testLogMessage(c *check.C, msg string, print func(string, ...field.M), fields ...field.M) map[string]interface{} {
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	var memLog bytes.Buffer
	log.SetOutput(&memLog)
	print(msg, fields...)
	var entry map[string]interface{}
	err := json.Unmarshal(memLog.Bytes(), &entry)
	c.Assert(err, check.IsNil)
	c.Assert(entry, check.NotNil)
	c.Assert(entry["msg"], check.Equals, msg)
	return entry
}

func (s *LogSuite) TestLogLevel(c *check.C) {
	err := os.Unsetenv(LevelEnvName)
	c.Assert(err, check.IsNil)
	initLogLevel()
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})

	var output bytes.Buffer
	log.SetOutput(&output)
	ctx := field.Context(context.Background(), "key", "value")
	var entry map[string]interface{}
	// Check if debug level log is printed when log level is info
	Debug().WithContext(ctx).Print("Testing debug level")
	err = json.Unmarshal(output.Bytes(), &entry)

	c.Assert(err, check.NotNil)
	c.Assert(output.String(), check.HasLen, 0)

	// Check if debug level log is printed when log level is debug
	err = os.Setenv(LevelEnvName, "debug")
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(LevelEnvName)
		c.Assert(err, check.IsNil)
		initLogLevel()
	}()
	initLogLevel()
	Debug().WithContext(ctx).Print("Testing debug level")

	cerr := json.Unmarshal(output.Bytes(), &entry)
	c.Assert(cerr, check.IsNil)
	c.Assert(entry, check.NotNil)
	c.Assert(entry["msg"], check.Equals, "Testing debug level")
}

func (s *LogSuite) TestCloneGlobalLogger(c *check.C) {
	hook := newTestLogHook()
	log.AddHook(hook)
	actual := cloneGlobalLogger()
	c.Assert(actual.Formatter, check.Equals, log.Formatter)
	c.Assert(actual.ReportCaller, check.Equals, log.ReportCaller)
	c.Assert(actual.Level, check.Equals, log.Level)
	c.Assert(actual.Out, check.Equals, log.Out)
	c.Assert(actual.Hooks, check.DeepEquals, log.Hooks)

	// changing `actual` should not affect global logger
	actual.SetFormatter(&logrus.TextFormatter{})
	actual.SetReportCaller(true)
	actual.SetLevel(logrus.ErrorLevel)
	actual.SetOutput(&bytes.Buffer{})
	actual.AddHook(&logHook{})

	c.Assert(actual.Formatter, check.Not(check.Equals), log.Formatter)
	c.Assert(actual.ReportCaller, check.Not(check.Equals), log.ReportCaller)
	c.Assert(actual.Level, check.Not(check.Equals), log.Level)
	c.Assert(actual.Out, check.Not(check.Equals), log.Out)
	c.Assert(actual.Hooks, check.Not(check.DeepEquals), log.Hooks)

	log.Println("Test message")
	c.Assert(len(hook.capturedMessages), check.Equals, 1)
	c.Assert(hook.capturedMessages[0].Message, check.Equals, "Test message")
}

func (s *LogSuite) TestSetFluentbitOutput(c *check.C) {
	for _, tc := range []struct {
		desc string
		url  *url.URL
		err  error
	}{
		{
			desc: "valid_url",
			url: &url.URL{
				Scheme: "tcp",
				Host:   "something",
			},
		},
		{
			desc: "path_is_set",
			url: &url.URL{
				Scheme: "tcp",
				Host:   "something",
				Path:   "something",
			},
			err: ErrPathSet,
		},
		{
			desc: "non_tcp_endpoint",
			url: &url.URL{
				Scheme: "http",
				Host:   "something",
				Path:   "something",
			},
			err: ErrNonTCPEndpoint,
		},
		{
			desc: "empty_endpoint",
			url:  &url.URL{},
			err:  ErrEndpointNotSet,
		},
		{
			desc: "nil_endpoint",
			err:  ErrEndpointNotSet,
		},
	} {
		err := SetFluentbitOutput(tc.url)
		c.Assert(err, check.Equals, tc.err)
	}
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
