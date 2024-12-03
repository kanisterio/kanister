package kanx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"maps"
	"sync"
)

const (
	tailTickDuration  = 3 * time.Second
	tempStdoutPattern = "kando.*.stdout"
	tempStderrPattern = "kando.*.stderr"
	streamBufferBytes = 4 * 1024 * 1024
)

type processServiceServer struct {
	UnimplementedProcessServiceServer
	mu               sync.Mutex
	processes        map[int64]*process
	outputDir        string
	tailTickDuration time.Duration
}

type process struct {
	mu       sync.Mutex
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
		processes:        map[int64]*process{},
		tailTickDuration: tailTickDuration,
	}
}

func (s *processServiceServer) CreateProcesses(_ context.Context, cpr *CreateProcessRequest) (*Process, error) {
	stdout, err := os.CreateTemp(s.outputDir, tempStdoutPattern)
	if err != nil {
		return nil, err
	}
	stderr, err := os.CreateTemp(s.outputDir, tempStderrPattern)
	if err != nil {
		return nil, err
	}
	// We use context.Background() here because the parameter ctx seems to get canceled when this returns.
	ctx, can := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, cpr.GetName(), cpr.GetArgs()...)
	p := &process{
		cmd:    cmd,
		doneCh: make(chan struct{}),
		stdout: stdout,
		stderr: stderr,
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
	s.mu.Lock()
	s.processes[int64(cmd.Process.Pid)] = p
	s.mu.Unlock()
	fields := field.M{"pid": cmd.Process.Pid, "stdout": stdout.Name(), "stderr": stderr.Name()}
	stdoutLogWriter.SetFields(fields)
	stderrLogWriter.SetFields(fields)
	log.Info().Print(processToProtoWithLock(p).String(), fields)
	go func() {
		err := p.cmd.Wait()
		p.err = err
		if exiterr, ok := err.(*exec.ExitError); ok {
			p.exitCode = exiterr.ExitCode()
		}
		err = stdout.Close()
		if err != nil {
			log.Error().WithError(err).Print("Failed to close stdout", fields)
		}
		err = stderr.Close()
		if err != nil {
			log.Error().WithError(err).Print("Failed to close stderr", fields)
		}
		can()
		close(p.doneCh)
		log.Info().Print(processToProtoWithLock(p).String())
	}()
	return &Process{
		Pid:   int64(cmd.Process.Pid),
		State: ProcessState_PROCESS_STATE_RUNNING,
	}, nil
}

func (s *processServiceServer) GetProcess(ctx context.Context, grp *GetProcessRequest) (*Process, error) {
	s.mu.Lock()
	q, ok := s.processes[grp.GetPid()]
	s.mu.Unlock()
	if !ok {
		return nil, errors.WithStack(errProcessNotFound)
	}
	ps := processToProtoWithLock(q)
	return ps, nil
}

func (s *processServiceServer) ListProcesses(lpr *ListProcessesRequest, lps ProcessService_ListProcessesServer) error {
	s.mu.Lock()
	processes := maps.Clone(s.processes)
	s.mu.Unlock()
	for _, p := range processes {
		ps := processToProtoWithLock(p)
		err := lps.Send(ps)
		if err != nil {
			return err
		}
	}
	return nil
}

var (
	errProcessNotFound = fmt.Errorf("Process not found")
)

func (s *processServiceServer) Stdout(por *ProcessOutputRequest, ss ProcessService_StdoutServer) error {
	s.mu.Lock()
	p, ok := s.processes[por.Pid]
	s.mu.Unlock()
	if !ok {
		return errors.WithStack(errProcessNotFound)
	}
	fh, err := os.Open(p.stdout.Name())
	if err != nil {
		return err
	}
	return s.streamOutput(ss, p, fh)
}

func (s *processServiceServer) Stderr(por *ProcessOutputRequest, ss ProcessService_StderrServer) error {
	s.mu.Lock()
	p, ok := s.processes[por.Pid]
	s.mu.Unlock()
	if !ok {
		return errors.WithStack(errProcessNotFound)
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
		buf.Reset()
	}
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

func processToProtoWithLock(p *process) *Process {
	p.mu.Lock()
	defer p.mu.Unlock()
	return processToProto(p)
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
