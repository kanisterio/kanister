package kanx

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func unixDialer(ctx context.Context, addr string) (net.Conn, error) {
	return net.Dial("unix", addr)
}

func newGRPCConnection(addr string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	opts = append(opts, grpc.WithContextDialer(unixDialer))
	// Add passthrough scheme if there is no scheme defined in the address
	u, err := url.Parse(addr)
	if err == nil && u.Scheme == "" {
		addr = "passthrough:///" + addr
	}
	return grpc.NewClient(addr, opts...)
}

func CreateProcess(ctx context.Context, addr string, name string, args []string) (*Process, error) {
	conn, err := newGRPCConnection(addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close() //nolint:errcheck
	c := NewProcessServiceClient(conn)
	in := &CreateProcessRequest{
		Name: name,
		Args: args,
	}
	return c.CreateProcess(ctx, in)
}

func GetProcess(ctx context.Context, addr string, pid int64) (*Process, error) {
	conn, err := newGRPCConnection(addr)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		}
	}()
	c := NewProcessServiceClient(conn)
	wpc, err := c.GetProcess(ctx, &ProcessPidRequest{
		Pid: pid,
	})
	if err != nil {
		return nil, err
	}
	return wpc, nil
}

func ListProcesses(ctx context.Context, addr string) ([]*Process, error) {
	conn, err := newGRPCConnection(addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close() //nolint:errcheck
	c := NewProcessServiceClient(conn)
	lpc, err := c.ListProcesses(ctx, &ListProcessesRequest{})
	if err != nil {
		return nil, err
	}
	ps := []*Process{}
	for {
		p, err := lpc.Recv()
		switch {
		case err == io.EOF:
			return ps, nil
		case err != nil:
			return nil, err
		}
		ps = append(ps, p)
	}
}

func SignalProcess(ctx context.Context, addr string, pid int64, signal int64) (*Process, error) {
	conn, err := newGRPCConnection(addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close() //nolint:errcheck
	c := NewProcessServiceClient(conn)
	wpc, err := c.SignalProcess(ctx, &SignalProcessRequest{
		Pid:    pid,
		Signal: signal,
	})
	if err != nil {
		return nil, err
	}
	return wpc, nil
}

func Stdout(ctx context.Context, addr string, pid int64, out io.Writer) error {
	conn, err := newGRPCConnection(addr)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck
	c := NewProcessServiceClient(conn)
	in := &ProcessPidRequest{
		Pid: pid,
	}
	poc, err := c.Stdout(ctx, in)
	if err != nil {
		return err
	}
	return output(ctx, out, poc)
}

func Stderr(ctx context.Context, addr string, pid int64, out io.Writer) error {
	conn, err := newGRPCConnection(addr)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck
	c := NewProcessServiceClient(conn)
	in := &ProcessPidRequest{
		Pid: pid,
	}
	poc, err := c.Stderr(ctx, in)
	if err != nil {
		return err
	}
	return output(ctx, out, poc)
}

type recver interface {
	Recv() (*Output, error)
}

func output(ctx context.Context, out io.Writer, sc recver) error {
	for {
		p, err := sc.Recv()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		}
		_, err = out.Write([]byte(p.Output))
		if err != nil {
			return err
		}
	}
}

type ProcessExitCode int

func (e ProcessExitCode) Error() string {
	return fmt.Sprintf("exit status %d", e)
}
