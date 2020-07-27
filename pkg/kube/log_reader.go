package kube

import (
	"context"
	"io"

	"k8s.io/client-go/rest"
)

var _ io.ReadCloser = (*logReader)(nil)

func newLogReader(rw rest.ResponseWrapper) *logReader {
	return &logReader{rw: rw}
}

// logReader defers a call to ResponseWrapper.Stream() until Read is called.
// This is to help handle kubernetes behavior where calls to
// Pod.GetLogs(...).Stream() will hang until at least one byte is returned.
// kubectl logs handles this in
// https://github.com/kubernetes/kubernetes/pull/67573/files#diff-12d472fe036bbe778e84dffc71564eb1R355
type logReader struct {
	rw  rest.ResponseWrapper
	err error
	rc  io.ReadCloser
}

func (lr *logReader) Read(p []byte) (n int, err error) {
	if lr.rc == nil {
		lr.rc, lr.err = lr.rw.Stream(context.TODO())
	}
	if lr.err != nil {
		return 0, lr.err
	}
	return lr.rc.Read(p)
}

func (lr *logReader) Close() error {
	if lr.rc == nil {
		return nil
	}
	return lr.rc.Close()
}
