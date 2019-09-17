package log

import (
	"context"

	"github.com/kanisterio/kanister/pkg/field"
)

type Logger interface {
	Print(msg string)
	WithContext(ctx context.Context, fields field.Fields) Logger
	WithError(err error) Logger
}
