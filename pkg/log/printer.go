package log

import (
	"context"
)

type Printer interface {
	Print(msg string)
	WithContext(ctx context.Context) Printer
	WithError(err error) Printer
}
