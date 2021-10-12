package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
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

// nolint
func (s *LogSuite) TestWithNilContext(c *C) {
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	// Should not panic
	WithContext(nil).Print("Message")
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
	os.Unsetenv(LevelEnvName)
	initLogLevel()
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})

	var output bytes.Buffer
	log.SetOutput(&output)
	ctx := field.Context(context.Background(), "key", "value")
	var entry map[string]interface{}
	//Check if debug level log is printed when log level is info
	Debug().WithContext(ctx).Print("Testing debug level")
	err := json.Unmarshal(output.Bytes(), &entry)

	c.Assert(err, NotNil)
	c.Assert(output.String(), HasLen, 0)

	//Check if debug level log is printed when log level is debug
	os.Setenv(LevelEnvName, "debug")
	defer func() {
		os.Unsetenv(LevelEnvName)
		initLogLevel()
	}()
	initLogLevel()
	Debug().WithContext(ctx).Print("Testing debug level")
	cerr := json.Unmarshal(output.Bytes(), &entry)
	c.Assert(cerr, IsNil)
	c.Assert(entry, NotNil)
	c.Assert(entry["msg"], Equals, "Testing debug level")
}
