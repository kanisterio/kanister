package kanx

import (
	"bytes"
	"context"
	"fmt"
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
)

const (
	tailTickDuration  = 3 * time.Second
	tempStdoutPattern = "kando.*.stdout"
	tempStderrPattern = "kando.*.stderr"
	streamBufferBytes = 4 * 1024 * 1024
)

type processServiceServer struct {
	UnimplementedProcessServiceServer
	processes        map[int64]*process
	outputDir        string
	tailTickDuration time.Duration
}

type process struct {
	cmd      *exec.Cmd
	doneCh   chan struct{}
	stdout   *os.File
	stderr   *os.File
	exitCode int
	err      error
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
	cmd.Stdout = p.stdout
	cmd.Stderr = p.stderr

	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	s.processes[int64(cmd.Process.Pid)] = p
	log.Info().Print(processToProto(p).String(), field.M{"stdout": stdout.Name(), "stderr": stderr.Name()})
	go func() {
		err := p.cmd.Wait()
		p.err = err
		if exiterr, ok := err.(*exec.ExitError); ok {
			p.exitCode = exiterr.ExitCode()
		}
		err = stdout.Close()
		if err != nil {
			log.Error().WithError(err).Print("Failed to close stdout", field.M{"pid": cmd.Process.Pid})
		}
		err = stderr.Close()
		if err != nil {
			log.Error().WithError(err).Print("Failed to close stderr", field.M{"pid": cmd.Process.Pid})
		}
		close(p.doneCh)
		log.Info().Print(processToProto(p).String())
	}()
	return &Process{
		Pid:   int64(cmd.Process.Pid),
		State: ProcessState_PROCESS_STATE_RUNNING,
	}, nil
}

func (s *processServiceServer) ListProcesses(lpr *ListProcessesRequest, lps ProcessService_ListProcessesServer) error {
	for _, p := range s.processes {
		ps := processToProto(p)
		err := lps.Send(ps)
		if err != nil {
			return err
		}
	}
	return nil
}

var errProcessNotFound = fmt.Errorf("Process not found")

func (s *processServiceServer) Stdout(por *ProcessOutputRequest, ss ProcessService_StdoutServer) error {
	p, ok := s.processes[por.Pid]
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
	p, ok := s.processes[por.Pid]
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
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case sig := <-stopChan:
			log.Info().Print("Gracefully stopping. Received Signal", field.M{"signal": sig})
		case <-ctx.Done():
			log.Info().Print("Gracefully stopping. Context canceled")
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
