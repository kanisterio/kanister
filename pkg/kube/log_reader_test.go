package kube

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	. "gopkg.in/check.v1"
	"k8s.io/client-go/rest"
)

type LogReaderSuite struct{}

var _ = Suite(&LogReaderSuite{})

var _ io.ReadCloser = (*buffer)(nil)

type buffer struct {
	*bytes.Buffer
}

func (b buffer) Close() error {
	return nil
}

var _ rest.ResponseWrapper = (*fakeResponseWrapper)(nil)

type fakeResponseWrapper struct {
	err error
	buf *bytes.Buffer
}

func (frw *fakeResponseWrapper) DoRaw() ([]byte, error) {
	return nil, nil
}
func (frw *fakeResponseWrapper) Stream() (io.ReadCloser, error) {
	return buffer{frw.buf}, frw.err
}

func (s *LogReaderSuite) TestLogReader(c *C) {
	err := fmt.Errorf("TEST")
	for _, tc := range []struct {
		rw  *fakeResponseWrapper
		err error
		out string
	}{
		{
			rw: &fakeResponseWrapper{
				err: nil,
				buf: bytes.NewBuffer(nil),
			},
			err: nil,
			out: "",
		},
		{
			rw: &fakeResponseWrapper{
				err: nil,
				buf: bytes.NewBuffer([]byte("foo")),
			},
			err: nil,
			out: "foo",
		},
		{
			rw: &fakeResponseWrapper{
				err: err,
				buf: nil,
			},
			err: err,
			out: "",
		},
		{
			rw: &fakeResponseWrapper{
				err: err,
				buf: bytes.NewBuffer([]byte("foo")),
			},
			err: err,
			out: "",
		},
	} {
		lr := newLogReader(tc.rw)
		out, err := ioutil.ReadAll(lr)
		c.Assert(err, Equals, tc.err)
		c.Assert(string(out), Equals, tc.out)
	}
}
