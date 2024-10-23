package kanx

import (
	"io"
	"sync"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

var _ io.Writer = (*logWriter)(nil)

type logWriter struct {
	logger log.Logger
	writer io.Writer
	fields field.M
	mutex  *sync.Mutex
}

func newLogWriter(l log.Logger, w io.Writer) *logWriter {
	return &logWriter{
		logger: l,
		writer: w,
		fields: nil,
		mutex:  &sync.Mutex{},
	}
}

func (lw *logWriter) SetFields(m field.M) {
	lw.mutex.Lock()
	defer lw.mutex.Unlock()
	lw.fields = m
}

func (lw *logWriter) Write(buf []byte) (int, error) {
	lw.mutex.Lock()
	f := lw.fields
	lw.mutex.Unlock()
	lw.logger.PrintTo(lw.writer, string(buf), f)
	return len(buf), nil
}
