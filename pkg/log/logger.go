package log

import (
	"context"

	"github.com/kanisterio/kanister/pkg/field"
)

type Logger interface {
	Print(msg string, fields ...field.M)
	WithContext(ctx context.Context) Logger
	WithError(err error) Logger
}
