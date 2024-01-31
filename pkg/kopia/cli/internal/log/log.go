package storage

import (
	"context"
	"io"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

// NopLogger is a logger that does nothing.
// TODO: Move to log package
type NopLogger struct{}

func (NopLogger) Print(msg string, fields ...field.M) {
}

func (NopLogger) PrintTo(w io.Writer, msg string, fields ...field.M) {
}

func (NopLogger) WithContext(ctx context.Context) log.Logger {
	return &NopLogger{}
}

func (NopLogger) WithError(err error) log.Logger {
	return &NopLogger{}
}
