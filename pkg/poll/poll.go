package poll

import (
	"context"
	"time"

	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
)

// Func returns true if the condition is satisfied, or an error if the loop
// should be aborted.
type Func func(context.Context) (bool, error)

// Wait calls WaitWithBackoff with default backoff parameters. The defaults are
// handled by the "github.com/jpillora/backoff" and are:
//   min = 100 * time.Millisecond
//   max = 10 * time.Second
//   factor = 2
//   jitter = false
func Wait(ctx context.Context, f Func) error {
	return WaitWithBackoff(ctx, backoff.Backoff{}, f)
}

// WaitWithBackoff calls a function until it returns true, an error, or until
// the context is done.
func WaitWithBackoff(ctx context.Context, b backoff.Backoff, f Func) error {
	for {
		if ok, err := f(ctx); err != nil || ok {
			return err
		}
		select {
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		default:
		}
		sleep := b.Duration()
		if deadline, ok := ctx.Deadline(); ok {
			ctxSleep := deadline.Sub(time.Now())
			sleep = minDuration(sleep, ctxSleep)
		}
		time.Sleep(sleep)
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
