package log

import (
	"context"
)

type Logger interface {
	Print(msg string)
	WithContext(ctx context.Context) Logger
	WithError(err error) Logger
}
