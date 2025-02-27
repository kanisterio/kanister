package kanx

import (
	"bytes"
	"context"
	"fmt"
	"hash/adler32"
	"os"
	"path"
	"sync"
	"syscall"
	"testing"
	"time"

	"google.golang.org/grpc"
	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/poll"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

const (
	ServerReadTimeout = 90 * time.Second
)

type KanXSuite struct{}

var _ = Suite(&KanXSuite{})

func tmpDir(c *C) string {
	// unix socket addresses typically cannot be longer than 100 characters
	// limit the size of the address while retaining some of the properties
	// of the original directory name
	hs := adler32.Checksum([]byte(c.TestName()))
	d, err := os.MkdirTemp("", fmt.Sprintf("%.20s%08x", c.TestName(), hs))
	c.Log("Directory: ", d)
	c.Assert(err, IsNil)
	return d
}

func newTestServer(dir string) *Server {
	var opts []grpc.ServerOption
	return &Server{
		grpcs: grpc.NewServer(opts...),
		pss: &processServiceServer{
			processes:        &sync.Map{},
			outputDir:        dir,
			tailTickDuration: time.Nanosecond,
		},
	}
}

func serverReady(ctx context.Context, addr string, c *C) {
	ctx, can := context.WithTimeout(ctx, ServerReadTimeout)
	defer can()
	for {
		select {
		case <-ctx.Done():
			c.Fatal("Timeout waiting for server to be ready")
			return
		default:
		}
		_, err := ListProcesses(ctx, addr)
		if err == nil {
			return
		}
	}
}

func (s *KanXSuite) TestServerCancellation(c *C) {
	d := tmpDir(c)
	addr := path.Join(d, "kanx.sock")
	ctx, can := context.WithCancel(context.Background())
	go func() {
		serverReady(ctx, addr, c)
		// test with context cancellation.  Cancel context as soon as its ready
		can()
	}()
	err := newTestServer(d).Serve(ctx, addr)
	c.Assert(err, IsNil)
}

func (s *KanXSuite) TestShortProcess(c *C) {
	d := tmpDir(c)
	addr := path.Join(d, "kanx.sock")
	ctx, can := context.WithCancel(context.Background())
	defer can()
	server := newTestServer(d)
	go func() {
		err := server.Serve(ctx, addr)
		c.Assert(err, IsNil)
	}()
	serverReady(ctx, addr, c)

	p, err := CreateProcess(ctx, addr, "echo", []string{"hello"})
	c.Assert(err, IsNil)
	c.Assert(p.GetPid(), Not(Equals), 0)
	c.Assert(p.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p.GetExitErr(), Equals, "")
	c.Assert(p.GetExitCode(), Equals, int64(0))

	// wait for termination
	_ = poll.Wait(ctx, func(context.Context) (bool, error) {
		p, err = GetProcess(ctx, addr, p.GetPid())
		c.Assert(err, IsNil)
		return p.GetState() != ProcessState_PROCESS_STATE_RUNNING, nil
	})

	// get the output from the coammnd.  stdout should contain "hello", stderr should be empty.
	buf := bytes.NewBuffer(make([]byte, 0, 1024*1024))
	err = Stdout(ctx, addr, p.GetPid(), buf)
	c.Assert(err, IsNil)
	c.Assert(buf.String(), Equals, "hello\n")

	buf.Reset()
	err = Stderr(ctx, addr, p.GetPid(), buf)
	c.Assert(err, IsNil)
	c.Assert(buf.String(), Equals, "")

	p0, err := GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_SUCCEEDED)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	// cleanup
	sp, ok := server.pss.loadProcess(p.GetPid())
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, Equals, os.ErrProcessDone)
}

func (s *KanXSuite) TestLongProcess(c *C) {
	d := tmpDir(c)
	addr := path.Join(d, "kanx.sock")
	ctx, can := context.WithCancel(context.Background())
	defer can()
	server := newTestServer(d)
	go func() {
		err := server.Serve(ctx, addr)
		c.Assert(err, IsNil)
	}()
	serverReady(ctx, addr, c)

	// run a process that will never terminate on its own
	p0, err := CreateProcess(ctx, addr, "tail", []string{"-f", "/dev/null"})
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Not(Equals), 0)
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	// signal that the process should be killed
	// get the internal process structure
	sp, ok := server.pss.loadProcess(p0.GetPid())
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, IsNil)
}

func (s *KanXSuite) TestGetProcess(c *C) {
	d := tmpDir(c)
	addr := path.Join(d, "kanx.sock")
	ctx, can := context.WithCancel(context.Background())
	defer can()
	server := newTestServer(d)
	go func() {
		err := server.Serve(ctx, addr)
		c.Assert(err, IsNil)
	}()
	serverReady(ctx, addr, c)

	p, err := CreateProcess(ctx, addr, "tail", []string{"-f", "/dev/null"})
	c.Assert(err, IsNil)
	c.Assert(p.GetPid(), Equals, p.GetPid())
	c.Assert(p.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p.GetExitErr(), Equals, "")
	c.Assert(p.GetExitCode(), Equals, int64(0))

	// test GetProcess
	p0, err := GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	// clean up
	sp, ok := server.pss.loadProcess(p.GetPid())
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, IsNil)
}

func (s *KanXSuite) TestSignalProcess_Int(c *C) {
	d := tmpDir(c)
	addr := path.Join(d, "kanx.sock")
	ctx, can := context.WithCancel(context.Background())
	defer can()
	server := newTestServer(d)
	go func() {
		err := server.Serve(ctx, addr)
		c.Assert(err, IsNil)
	}()
	serverReady(ctx, addr, c)

	p, err := CreateProcess(ctx, addr, "sleep", []string{"1s"})
	c.Assert(err, IsNil)

	// test SignalProcess, SIGINT
	p0, err := SignalProcess(ctx, addr, p.GetPid(), int64(syscall.SIGINT))
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	// wait for termination
	_ = poll.Wait(ctx, func(context.Context) (bool, error) {
		p0, err = GetProcess(ctx, addr, p0.GetPid())
		c.Assert(err, IsNil)
		return p0.GetState() != ProcessState_PROCESS_STATE_RUNNING, nil
	})

	// get final process state
	p0, err = GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_FAILED)
	c.Assert(p0.GetExitErr(), Equals, "signal: interrupt")
	c.Assert(p0.GetExitCode(), Equals, int64(-1))

	// clean up
	sp, ok := server.pss.loadProcess(p.GetPid())
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, Equals, os.ErrProcessDone)
}

// TestSignalProcess_Stp don't assume that a signal leads to a process termination.
func (s *KanXSuite) TestSignalProcess_Stp(c *C) {
	d := tmpDir(c)
	addr := path.Join(d, "kanx.sock")
	ctx, can := context.WithCancel(context.Background())
	defer can()
	server := newTestServer(d)
	go func() {
		err := server.Serve(ctx, addr)
		c.Assert(err, IsNil)
	}()
	serverReady(ctx, addr, c)

	p, err := CreateProcess(ctx, addr, "sleep", []string{"2s"})
	c.Assert(err, IsNil)

	// test SignalProcess, SIGSTOP
	p0, err := SignalProcess(ctx, addr, p.GetPid(), int64(syscall.SIGSTOP))
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	// test SignalProcess, SIGCONT
	p0, err = SignalProcess(ctx, addr, p.GetPid(), int64(syscall.SIGCONT))
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	// wait for termination
	_ = poll.Wait(ctx, func(context.Context) (bool, error) {
		p0, err = GetProcess(ctx, addr, p0.GetPid())
		c.Assert(err, IsNil)
		return p0.GetState() != ProcessState_PROCESS_STATE_RUNNING, nil
	})

	// get final process state
	p0, err = GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_SUCCEEDED)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	// clean up
	sp, ok := server.pss.loadProcess(p.GetPid())
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, Equals, os.ErrProcessDone)
}

// TestSignalProcess_Kill check process state after a SIGKILL signal has been sent.
// this should result in an error in the process structure and a client error stating
// that the process was killed
func (s *KanXSuite) TestSignalProcess_Kill(c *C) {
	d := tmpDir(c)
	addr := path.Join(d, "kanx.sock")
	ctx, can := context.WithCancel(context.Background())
	defer can()
	server := newTestServer(d)
	go func() {
		err := server.Serve(ctx, addr)
		c.Assert(err, IsNil)
	}()
	serverReady(ctx, addr, c)

	p, err := CreateProcess(ctx, addr, "sleep", []string{"2s"})
	c.Assert(err, IsNil)

	// test SignalProcess, SIGKILL
	p0, err := SignalProcess(ctx, addr, p.GetPid(), int64(syscall.SIGKILL))
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	// wait for termination
	_ = poll.Wait(ctx, func(context.Context) (bool, error) {
		p0, err = GetProcess(ctx, addr, p0.GetPid())
		c.Assert(err, IsNil)
		return p0.GetState() != ProcessState_PROCESS_STATE_RUNNING, nil
	})

	// get final process state
	p0, err = GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_FAILED)
	c.Assert(p0.GetExitErr(), Equals, "signal: killed")
	c.Assert(p0.GetExitCode(), Equals, int64(-1))

	// clean up
	sp, ok := server.pss.loadProcess(p.GetPid())
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, Equals, os.ErrProcessDone)
}

func (s *KanXSuite) TestCreateProcess_Exit2(c *C) {
	d := tmpDir(c)
	addr := path.Join(d, "kanx.sock")
	ctx, can := context.WithCancel(context.Background())
	defer can()
	server := newTestServer(d)
	go func() {
		err := server.Serve(ctx, addr)
		c.Assert(err, IsNil)
	}()
	serverReady(ctx, addr, c)

	p0, err := CreateProcess(ctx, addr, "bash", []string{"-c", "exit 2"})
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Not(Equals), 0)
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	_ = poll.Wait(ctx, func(context.Context) (bool, error) {
		p0, err = GetProcess(ctx, addr, p0.GetPid())
		c.Assert(err, IsNil)
		return p0.GetState() != ProcessState_PROCESS_STATE_RUNNING, nil
	})

	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_FAILED)
	c.Assert(p0.GetExitErr(), Equals, "exit status 2")
	c.Assert(p0.GetExitCode(), Equals, int64(2))
}

type countWriter struct {
	C     *C
	Count int64
}

func (w *countWriter) Write(p []byte) (int, error) {
	l := len(p)
	w.Count += int64(l)
	return l, nil
}

func (s *KanXSuite) TestCreateProcess_BufferOverflow_1(c *C) {
	c.Skip("Buffer Overflow Test being skipped due to excessive logging.  issue #3386")
	d := tmpDir(c)
	addr := path.Join(d, "kanx.sock")
	ctx, can := context.WithCancel(context.Background())
	defer can()
	server := newTestServer(d)
	go func() {
		err := server.Serve(ctx, addr)
		c.Assert(err, IsNil)
	}()
	serverReady(ctx, addr, c)

	p0, err := CreateProcess(ctx, addr, "/bin/bash", []string{"-c", "yes | dd bs=8192 count=1024"})
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Not(Equals), 0)
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	_ = poll.Wait(ctx, func(context.Context) (bool, error) {
		p0, err = GetProcess(ctx, addr, p0.GetPid())
		c.Assert(err, IsNil)
		return p0.GetState() != ProcessState_PROCESS_STATE_RUNNING, nil
	})

	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_SUCCEEDED)
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	cw := &countWriter{C: c}
	err = Stdout(ctx, addr, p0.GetPid(), cw)
	c.Assert(err, IsNil)
	c.Assert(cw.Count, Equals, int64(1024*8192))

	// Conditions that will exist if the gRPC buffer overflows:
	//
	//	c.Assert(err, NotNil)
	//	c.Assert(err.Error(), Matches, ".*received message larger than max.*")
}
