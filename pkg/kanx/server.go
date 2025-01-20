package kanx

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kanisterio/errkit"
	"google.golang.org/grpc"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	tailTickDuration  = 3 * time.Second
	tempStdoutPattern = "kando.*.stdout"
	tempStderrPattern = "kando.*.stderr"
	streamBufferBytes = 4 * 1024 * 1024
)

type processServiceServer struct {
	UnimplementedProcessServiceServer
	processes        *sync.Map
	outputDir        string
	tailTickDuration time.Duration
}

type process struct {
	// many reads on process data and only a write on process exit - use RWMutex.
	// minimal risk of reads blocking writes.
	mu       *sync.RWMutex
	cmd      *exec.Cmd
	doneCh   chan struct{}
	stdout   *os.File
	stderr   *os.File
	exitCode int
	err      error
	fault    error
	cancel   context.CancelFunc
}

func newProcessServiceServer() *processServiceServer {
	return &processServiceServer{
		processes:        &sync.Map{},
		tailTickDuration: tailTickDuration,
	}
}

func (s *processServiceServer) CreateProcess(_ context.Context, cpr *CreateProcessRequest) (*Process, error) {
	stdoutfd, err := os.CreateTemp(s.outputDir, tempStdoutPattern)
	if err != nil {
		return nil, err
	}
	stderrfd, err := os.CreateTemp(s.outputDir, tempStderrPattern)
	if err != nil {
		return nil, err
	}
	// We use context.Background() here because the parameter ctx seems to get canceled when this returns.
	ctx, can := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, cpr.GetName(), cpr.GetArgs()...)
	p := &process{
		mu:     &sync.RWMutex{},
		cmd:    cmd,
		doneCh: make(chan struct{}),
		stdout: stdoutfd,
		stderr: stderrfd,
		cancel: can,
	}
	stdoutLogWriter := newLogWriter(log.Info(), os.Stdout)
	stderrLogWriter := newLogWriter(log.Info(), os.Stderr)
	cmd.Stdout = io.MultiWriter(p.stdout, stdoutLogWriter)
	cmd.Stderr = io.MultiWriter(p.stderr, stderrLogWriter)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	s.storeProcess(int64(cmd.Process.Pid), p)
	fields := field.M{"pid": cmd.Process.Pid, "stdout": stdoutfd.Name(), "stderr": stderrfd.Name()}
	stdoutLogWriter.SetFields(fields)
	stderrLogWriter.SetFields(fields)
	log.Info().Print(processToProto(p).String(), fields)
	// one goroutine in server per forked process.  link between pid and output files will be lost
	// if &process structure is lost.
	go func() {
		// wait until process is finished.  do not use lock as there may be readers or writers
		// on p and cmd is not expected to change (the state in cmd is system managed)
		err := p.cmd.Wait()
		// possible readers concurrent to write: lock the p structure for exit status update.
		// keep the lock period as short as possible.  remove the possibility of blocking
		// on log writes by moving them outside the region of the lock.
		// go doesn't have lock promotion, so there's a small gap here from when Wait finishes
		// until acquiring a write lock.
		p.mu.Lock()
		p.err = err
		if exiterr, ok := err.(*exec.ExitError); ok {
			p.exitCode = exiterr.ExitCode()
		}
		// no action will be taken on close errors, so just save the errors for logging
		// later
		stdoutErr := stdoutfd.Close()
		stderrErr := stderrfd.Close()
		can()
		close(p.doneCh)
		prc := processToProto(p)
		p.mu.Unlock()
		if stdoutErr != nil {
			log.Error().WithError(err).Print("Failed to close stdout", fields)
		}
		if stderrErr != nil {
			log.Error().WithError(err).Print("Failed to close stderr", fields)
		}
		log.Info().Print(prc.String())
	}()
	return &Process{
		Pid:   int64(cmd.Process.Pid),
		State: ProcessState_PROCESS_STATE_RUNNING,
	}, nil
}

func (s *processServiceServer) loadProcess(pid int64) (*process, bool) {
	v, ok := s.processes.Load(pid)
	if !ok {
		return nil, false
	}
	return v.(*process), true
}

func (s *processServiceServer) storeProcess(pid int64, p *process) {
	s.processes.Store(pid, p)
}

func (s *processServiceServer) GetProcess(_ context.Context, grp *ProcessPidRequest) (*Process, error) {
	p, ok := s.loadProcess(grp.GetPid())
	if !ok {
		return nil, errkit.WithStack(errProcessNotFound)
	}
	return processToProtoWithLock(p), nil
}

func (s *processServiceServer) SignalProcess(_ context.Context, grp *SignalProcessRequest) (*Process, error) {
	p, ok := s.loadProcess(grp.GetPid())
	if !ok {
		return nil, errkit.WithStack(errProcessNotFound)
	}
	// low level signal call
	syssig := syscall.Signal(grp.Signal)
	p.mu.Lock()
	defer p.mu.Unlock()
	err := p.cmd.Process.Signal(syssig)
	if err != nil {
		// `fault` tracks IPC errors
		p.fault = err
	}
	return processToProto(p), err
}

func (s *processServiceServer) ListProcesses(_ *ListProcessesRequest, lps ProcessService_ListProcessesServer) error {
	var err error
	s.processes.Range(func(key, value any) bool {
		err = lps.Send(processToProtoWithLock(value.(*process)))
		return err == nil
	})
	return err
}

var errProcessNotFound = errkit.NewSentinelErr("Process not found")

func (s *processServiceServer) Stdout(por *ProcessPidRequest, ss ProcessService_StdoutServer) error {
	p, ok := s.loadProcess(por.Pid)
	if !ok {
		return errkit.WithStack(errProcessNotFound)
	}
	fh, err := os.Open(p.stdout.Name())
	if err != nil {
		return err
	}
	return s.streamOutput(ss, p, fh)
}

func (s *processServiceServer) Stderr(por *ProcessPidRequest, ss ProcessService_StderrServer) error {
	p, ok := s.loadProcess(por.Pid)
	if !ok {
		return errkit.WithStack(errProcessNotFound)
	}
	fh, err := os.Open(p.stderr.Name())
	if err != nil {
		return err
	}
	return s.streamOutput(ss, p, fh)
}

type sender interface {
	Send(*Output) error
}

func (s *processServiceServer) streamOutput(ss sender, p *process, fh *os.File) error {
	buf := bytes.NewBuffer(make([]byte, 0, streamBufferBytes)) // 4MiB is the max size of a GRPC request
	t := time.NewTicker(s.tailTickDuration)
	for {
		n, err := buf.ReadFrom(fh)
		switch {
		case err != nil:
			return err
		case n == 0:
			select {
			case <-p.doneCh:
				return nil
			default:
			}
			<-t.C
			continue
		}
		o := &Output{Output: buf.String()}
		err = ss.Send(o)
		if err != nil {
			return err
		}
	}
}

func processToProtoWithLock(p *process) *Process {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return processToProto(p)
}

func processToProto(p *process) *Process {
	ps := &Process{
		Pid: int64(p.cmd.Process.Pid),
	}
	select {
	case <-p.doneCh:
		ps.State = ProcessState_PROCESS_STATE_SUCCEEDED
		if p.err != nil {
			ps.State = ProcessState_PROCESS_STATE_FAILED
			ps.ExitErr = p.err.Error()
			ps.ExitCode = int64(p.exitCode)
		}
	default:
		ps.State = ProcessState_PROCESS_STATE_RUNNING
	}
	return ps
}

type Server struct {
	grpcs *grpc.Server
	pss   *processServiceServer
}

func NewServer() *Server {
	var opts []grpc.ServerOption
	return &Server{
		grpcs: grpc.NewServer(opts...),
		pss:   newProcessServiceServer(),
	}
}

func (s *Server) Serve(ctx context.Context, addr string) error {
	ctx, can := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer can()
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err == context.Canceled {
			log.Info().Print("Gracefully stopping. Parent context canceled")
		} else {
			log.Info().WithError(err).Print("Gracefully stopping.")
		}
		s.grpcs.GracefulStop()
	}()
	RegisterProcessServiceServer(s.grpcs, s.pss)
	lis, err := net.Listen("unix", addr)
	if err != nil {
		return err
	}
	log.Info().Print("Listening on socket", field.M{"address": lis.Addr()})
	defer os.Remove(addr) //nolint:errcheck
	return s.grpcs.Serve(lis)
}
