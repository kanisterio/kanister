package log

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	defaultEntryBufferCount = 64
	defaultLogBufferSize    = 4 * 1024
	defaultPushTime         = 500 * time.Millisecond
	defaultConnTimeout      = 500 * time.Millisecond
)

// FluentbitHook is async Logrus hook for Fluentbit.
// It sends JSON encoding of the log entries over TCP connection.
type FluentbitHook struct {
	logs chan *logrus.Entry
}

// NewFluentbitHook creates a non-blocking Logrus hook
// which sends JSON logs to Fluentbit over TCP.
//
//	h := NewFluentbithook("X.Y.Z.W:12345")
//	logrus.AddHook(h)
func NewFluentbitHook(endpoint string) *FluentbitHook {
	ec := make(chan *logrus.Entry, defaultEntryBufferCount)

	go func(in <-chan *logrus.Entry) {
		buff := new(bytes.Buffer)
		buff.Grow(defaultLogBufferSize)
		t := time.NewTimer(defaultPushTime)
		for {
			select {
			case e := <-in:
				buff.Write(entryToJSON(e))
				if buff.Len() >= defaultLogBufferSize {
					handleBuffer(buff, endpoint)
					if !t.Stop() {
						<-t.C
					}
					t.Reset(defaultPushTime)
				}
			case <-t.C:
				if buff.Len() != 0 {
					handleBuffer(buff, endpoint)
				}
				t.Reset(defaultPushTime)
			}
		}
	}(ec)

	return &FluentbitHook{logs: ec}
}

// dial establishes TCP connection with endpoint and
// sets keep alive to true.
func dial(endpoint string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", endpoint, defaultConnTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "Fluentbit connection problem")
	}
	return conn, nil
}

// handle will open connection and sends the message through it.
func handle(msgs []byte, endpoint string) error {
	conn, err := dial(endpoint)
	if err != nil {
		return errors.Wrap(err, "Fluentbit connection error")
	}
	defer conn.Close() //nolint:errcheck
	_, err = conn.Write(msgs)
	if err != nil {
		return errors.Wrap(err, "Fluentbit write error")
	}
	return nil
}

// handleBuffer passes buffer to `handle` and emits an error message
// in case it fails. In the end, the buffer is zeroed.
func handleBuffer(buff *bytes.Buffer, endpoint string) {
	if err := handle(buff.Bytes(), endpoint); err != nil {
		fmt.Fprintln(os.Stderr, "Log message dropped (buffer):", buff.String(), "Error:", err)
	}
	buff.Reset()
}

// Fire sends log entry to Fluentbit instance asyncronously.
func (f *FluentbitHook) Fire(e *logrus.Entry) error {
	select {
	case f.logs <- e:
	default:
		fmt.Fprintln(os.Stderr, "Log message dropped (channel):", e)
	}
	return nil
}

// Levels returns all level log levels to indicate
// Logrus that this hook wants to receive all logs.
func (f *FluentbitHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
