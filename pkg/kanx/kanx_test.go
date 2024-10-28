package kanx

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"sync"
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
			processes:        map[int64]*process{},
			outputDir:        dir,
			tailTickDuration: time.Nanosecond,
		},
	}
}

func serverReady(ctx context.Context, addr string, c *C) {
	ctx, can := context.WithTimeout(ctx, 10*time.Second)
	defer can()
	for {
		select {
		case <-ctx.Done():
			c.Error("Timeout waiting for server to be ready")
			c.Fail()
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

	ps, err := ListProcesses(ctx, addr)
	c.Assert(err, IsNil)
	c.Assert(ps, HasLen, 1)
	c.Assert(ps[0].GetPid(), Equals, p.GetPid())
	c.Assert(ps[0].GetState(), Equals, ProcessState_PROCESS_STATE_SUCCEEDED)
	c.Assert(ps[0].GetExitErr(), Equals, "")
	c.Assert(ps[0].GetExitCode(), Equals, int64(0))
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

	ps, err := ListProcesses(ctx, addr)
	c.Assert(err, IsNil)
	c.Assert(ps, HasLen, 1)
	c.Assert(ps[0].GetPid(), Equals, p.GetPid())
	c.Assert(ps[0].GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(ps[0].GetExitErr(), Equals, "")
	c.Assert(ps[0].GetExitCode(), Equals, int64(0))

	ctx = context.Background()
	buf := bytes.NewBuffer(make([]byte, 0, 1024*1024))
	isCancelled := false
	go func() {
		err := Stdout(ctx, addr, p.GetPid(), buf)
		c.Assert(err, IsNil)
		c.Assert(buf.String(), Equals, "")
		c.Assert(isCancelled, Equals, true)
	}()
	sp, ok := server.pss.processes[p.GetPid()]
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

	ps, err := ListProcesses(ctx, addr)
	c.Assert(err, IsNil)
	c.Assert(ps, HasLen, 1)
	c.Assert(ps[0].GetPid(), Equals, p.GetPid())
	c.Assert(ps[0].GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(ps[0].GetExitErr(), Equals, "")
	c.Assert(ps[0].GetExitCode(), Equals, int64(0))

	sp, ok := server.pss.processes[p.GetPid()]
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, IsNil)

	_ = poll.Wait(ctx, func(context.Context) (bool, error) {
		ps, err = ListProcesses(ctx, addr)
		c.Assert(err, IsNil)
		c.Assert(ps, HasLen, 1)
		return ps[0].GetState() != ProcessState_PROCESS_STATE_RUNNING, nil
	})
	c.Assert(ps[0].GetPid(), Equals, p.GetPid())
	c.Assert(ps[0].GetState(), Equals, ProcessState_PROCESS_STATE_FAILED)
	c.Assert(ps[0].GetExitErr(), Equals, "signal: killed")
	c.Assert(ps[0].GetExitCode(), Equals, int64(-1))

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

	ps, err := ListProcesses(ctx, addr)
	c.Assert(err, IsNil)
	c.Assert(ps, HasLen, 1)
	c.Assert(ps[0].GetPid(), Equals, p.GetPid())
	c.Assert(ps[0].GetState(), Equals, ProcessState_PROCESS_STATE_RUNNING)
	c.Assert(ps[0].GetExitErr(), Equals, "")
	c.Assert(ps[0].GetExitCode(), Equals, int64(0))

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

	sp, ok := server.pss.processes[p.GetPid()]
	c.Assert(ok, Equals, true)
	err = sp.cmd.Process.Kill()
	c.Assert(err, IsNil)

	_ = poll.Wait(ctx, func(context.Context) (bool, error) {
		ps, err = ListProcesses(ctx, addr)
		c.Assert(err, IsNil)
		c.Assert(ps, HasLen, 1)
		return ps[0].GetState() != ProcessState_PROCESS_STATE_RUNNING, nil
	})

	c.Assert(ps[0].GetPid(), Equals, p.GetPid())
	c.Assert(ps[0].GetState(), Equals, ProcessState_PROCESS_STATE_FAILED)
	c.Assert(ps[0].GetExitErr(), Equals, "signal: killed")
	c.Assert(ps[0].GetExitCode(), Equals, int64(-1))
}
