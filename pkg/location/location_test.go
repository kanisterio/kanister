package location

import (
	"bytes"
	"context"
	"io"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type LocationSuite struct{}

var _ = Suite(&LocationSuite{})

func (s *LocationSuite) TestWrite(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		in      io.Reader
		bin     string
		args    []string
		env     []string
		checker Checker
	}{
		{
			in:      bytes.NewBufferString("hello"),
			bin:     "",
			args:    nil,
			env:     nil,
			checker: NotNil,
		},
		{
			in:      bytes.NewBufferString("hello"),
			bin:     "cat",
			args:    nil,
			env:     nil,
			checker: IsNil,
		},
		{
			in:      bytes.NewBufferString("echo hello"),
			bin:     "bash",
			args:    nil,
			env:     nil,
			checker: IsNil,
		},
		{
			in:      bytes.NewBufferString("INVALID"),
			bin:     "bash",
			args:    nil,
			env:     nil,
			checker: NotNil,
		},
	} {

		err := writeExec(ctx, tc.in, tc.bin, tc.args, tc.env)
		c.Check(err, tc.checker)
	}
}

func (s *LocationSuite) TestRead(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		out     string
		bin     string
		args    []string
		env     []string
		checker Checker
	}{
		{
			out:     "",
			bin:     "",
			args:    nil,
			env:     nil,
			checker: NotNil,
		},
		{
			out:     "",
			bin:     "echo",
			args:    []string{"-n"},
			env:     nil,
			checker: IsNil,
		},
		{
			out:     "hello",
			bin:     "echo",
			args:    []string{"-n", "hello"},
			env:     nil,
			checker: IsNil,
		},
	} {
		buf := bytes.NewBuffer(nil)
		err := readExec(ctx, buf, tc.bin, tc.args, tc.env)
		c.Check(err, tc.checker)
		c.Check(buf.String(), Equals, tc.out)
	}
}
