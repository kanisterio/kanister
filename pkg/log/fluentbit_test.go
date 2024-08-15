package log

import (
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

const (
	numMsgs      = 256
	fakeEndPoint = "127.0.0.1:25000"
)

type FluentbitSuite struct{}

var _ = Suite(&FluentbitSuite{})

func (s *FluentbitSuite) TestSendLogsServerRunning(c *C) {
	end := make(chan bool)
	// Fake Fluentbit
	go runServer(0, end, c)

	h := NewFluentbitHook(fakeEndPoint)
	// Syncronize push frequency with the processing frequency of the hook.
	// Assuming that it will minimize the number of the logs dropped.
	_ = pushMultipleLogs(h, numMsgs, defaultEntryBufferCount, defaultPushTime, 0)

	for range end {
	}
}

func (s *FluentbitSuite) TestSendLogsServerFailedInTheMiddle(c *C) {
	c.Logf("Error messages are expected in this test")

	end := make(chan bool)
	go runServer(numMsgs/2, end, c)

	h := NewFluentbitHook(fakeEndPoint)
	_ = pushMultipleLogs(h, numMsgs, defaultEntryBufferCount, defaultPushTime, 0)

	for range end {
	}
}

func (s *FluentbitSuite) TestSendLogsServerUnavailableFromStart(c *C) {
	c.Logf("Error messages are expected in this test")

	h := NewFluentbitHook(fakeEndPoint)

	waitFor := 10 * time.Second
	if ok := pushMultipleLogs(h, numMsgs, defaultEntryBufferCount, defaultPushTime, waitFor); !ok {
		c.Logf("Hook is stuck")
		c.Fail()
	}
}

func runServer(failAfterNLogs int, endFlag chan<- bool, c *C) {
	result := make([]string, 0)
	t := time.NewTimer(10 * time.Second)
	defer close(endFlag)

	l := resolveAndListen(fakeEndPoint, c)
	defer func() {
		err := l.Close()
		c.Assert(err, IsNil)
	}()

Loop:
	for {
		select {
		case <-t.C:
			// Success condition is that the server
			// processed at least 1 message in 5 sec
			c.Assert(result, Not(HasLen), 0)
			break Loop
		default:
			// or it processed all of them under 5 sec.
			if len(result) == numMsgs {
				break Loop
			}
		}
		_ = l.SetDeadline(time.Now().Add(2 * time.Second))
		conn, aerr := l.Accept()
		if aerr != nil {
			continue
		}
		bytes, rerr := io.ReadAll(conn)
		c.Assert(rerr, IsNil)

		strs := strings.Split(strings.Trim(string(bytes), "\n"), "\n")
		result = append(result, strs...)
		c.Assert(conn.Close(), IsNil)
		if failAfterNLogs != 0 && len(result) > failAfterNLogs {
			c.Logf("Server is failed as expected after %d logs", failAfterNLogs)
			break
		}
	}
	c.Logf("Server: Received %d of total %d logs", len(result), numMsgs)
}

func resolveAndListen(endpoint string, c *C) *net.TCPListener {
	addr, err := net.ResolveTCPAddr("tcp", endpoint)
	c.Assert(err, IsNil)
	l, err := net.ListenTCP("tcp", addr)
	c.Assert(err, IsNil)
	return l
}

func pushMultipleLogs(hook *FluentbitHook, logsAmount int, sleepAfterNLogs int, sleepDuration time.Duration, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for i := 0; i < logsAmount; i++ {
		e := logrus.NewEntry(nil).WithField(strconv.Itoa(i), i)
		e.Level = logrus.InfoLevel
		_ = hook.Fire(e)
		if timeout != 0 && deadline.Before(time.Now()) {
			return false
		}
		if i%sleepAfterNLogs == 0 {
			time.Sleep(sleepDuration)
		}
	}
	return true
}
