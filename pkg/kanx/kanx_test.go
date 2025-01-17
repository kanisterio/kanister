package kanx

import (
	"bytes"
	"context"
	"io"
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

type KanXSuite struct{}

var _ = Suite(&KanXSuite{})

func tmpDir(c *C) string {
	d, err := os.MkdirTemp("", c.TestName())
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
	ctx, can := context.WithTimeout(ctx, 90*time.Second)
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
	go func() {
		err := newTestServer(d).Serve(ctx, addr)
		c.Assert(err, IsNil)
	}()
	serverReady(ctx, addr, c)

	p, err := CreateProcess(ctx, addr, "echo", []string{"hello"})
	c.Assert(err, IsNil)
	c.Assert(p.GetPid(), Not(Equals), 0)
	c.Assert(p.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p.GetExitErr(), Equals, "")
	c.Assert(p.GetExitCode(), Equals, int64(0))

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

	p, err := CreateProcess(ctx, addr, "tail", []string{"-f", "/dev/null"})
	c.Assert(err, IsNil)
	c.Assert(p.GetPid(), Not(Equals), 0)
	c.Assert(p.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p.GetExitErr(), Equals, "")
	c.Assert(p.GetExitCode(), Equals, int64(0))

	p0, err := GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	ctx = context.Background()
	buf := bytes.NewBuffer(make([]byte, 0, 1024*1024))
	isCancelled := false
	go func() {
		err := Stdout(ctx, addr, p.GetPid(), buf)
		c.Assert(err, IsNil)
		c.Assert(buf.String(), Equals, "")
		c.Assert(isCancelled, Equals, true)
	}()
	sp, ok := server.pss.loadProcess(p.GetPid())
	c.Assert(ok, Equals, true)
	isCancelled = true
	err = sp.cmd.Process.Kill()
	c.Assert(err, IsNil)

	buf.Reset()
	err = Stdout(ctx, addr, p.GetPid(), buf)
	c.Assert(err, IsNil)
	c.Assert(buf.String(), Equals, "")

	buf.Reset()
	err = Stderr(ctx, addr, p.GetPid(), buf)
	c.Assert(err, IsNil)
	c.Assert(buf.String(), Equals, "")
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
	err = Stderr(ctx, addr, p.GetPid(), io.Discard)
	c.Assert(err, IsNil)

	// get final process state
	p0, err = GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_FAILED)
	c.Assert(p0.GetExitErr(), Equals, "signal: interrupt")
	c.Assert(p0.GetExitCode(), Equals, int64(-1))

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

	p, err := CreateProcess(ctx, addr, "sleep", []string{"1s"})
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
	err = Stderr(ctx, addr, p.GetPid(), io.Discard)
	c.Assert(err, IsNil)

	// get final process state
	p0, err = GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_SUCCEEDED)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	sp, ok := server.pss.loadProcess(p.GetPid())
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, Equals, os.ErrProcessDone)
}

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

	p, err := CreateProcess(ctx, addr, "sleep", []string{"1s"})
	c.Assert(err, IsNil)

	// test SignalProcess, SIGKILL
	p0, err := SignalProcess(ctx, addr, p.GetPid(), int64(syscall.SIGKILL))
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	// wait for termination
	err = Stderr(ctx, addr, p.GetPid(), io.Discard)
	c.Assert(err, IsNil)

	// get final process state
	p0, err = GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_FAILED)
	c.Assert(p0.GetExitErr(), Equals, "signal: killed")
	c.Assert(p0.GetExitCode(), Equals, int64(-1))

	sp, ok := server.pss.loadProcess(p.GetPid())
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, Equals, os.ErrProcessDone)
}

func (s *KanXSuite) TestError(c *C) {
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
	c.Assert(p.GetPid(), Not(Equals), 0)
	c.Assert(p.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p.GetExitErr(), Equals, "")
	c.Assert(p.GetExitCode(), Equals, int64(0))

	p0, err := GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	sp, ok := server.pss.loadProcess(p.GetPid())
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, IsNil)

	_ = poll.Wait(ctx, func(context.Context) (bool, error) {
		p0, err = GetProcess(ctx, addr, p.GetPid())
		c.Assert(err, IsNil)
		return p0.GetState() != ProcessState_PROCESS_STATE_RUNNING, nil
	})

	// test error details from GetProcesses
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_FAILED)
	c.Assert(p0.GetExitErr(), Equals, "signal: killed")
	c.Assert(p0.GetExitCode(), Equals, int64(-1))

	buf := bytes.NewBuffer(make([]byte, 0, 1024*1024))
	err = Stdout(ctx, addr, p.GetPid(), buf)
	c.Assert(err, IsNil)
	c.Assert(buf.String(), Equals, "")

	buf.Reset()
	err = Stderr(ctx, addr, p.GetPid(), buf)
	c.Assert(err, IsNil)
	c.Assert(buf.String(), Equals, "")
}

func (s *KanXSuite) TestParallelStdout(c *C) {
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

	p, err := CreateProcess(ctx, addr, "yes", nil)
	c.Assert(err, IsNil)
	c.Assert(p.GetPid(), Not(Equals), 0)
	c.Assert(p.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p.GetExitErr(), Equals, "")
	c.Assert(p.GetExitCode(), Equals, int64(0))

	p0, err := GetProcess(ctx, addr, p.GetPid())
	c.Assert(err, IsNil)
	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(p0.GetExitErr(), Equals, "")
	c.Assert(p0.GetExitCode(), Equals, int64(0))

	nw := io.Discard
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	for range make([]struct{}, 100) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = Stdout(ctx, addr, p.GetPid(), nw)
			c.Assert(err, IsNil)
			err = Stderr(ctx, addr, p.GetPid(), nw)
			c.Assert(err, IsNil)
		}()
	}

	sp, ok := server.pss.loadProcess(p.GetPid())
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, IsNil)

	_ = poll.Wait(ctx, func(context.Context) (bool, error) {
		p0, err = GetProcess(ctx, addr, p.GetPid())
		c.Assert(err, IsNil)
		return p0.GetState() != ProcessState_PROCESS_STATE_RUNNING, nil
	})

	c.Assert(p0.GetPid(), Equals, p.GetPid())
	c.Assert(p0.GetState(), Equals, ProcessState_PROCESS_STATE_FAILED)
	c.Assert(p0.GetExitErr(), Equals, "signal: killed")
	c.Assert(p0.GetExitCode(), Equals, int64(-1))
}
