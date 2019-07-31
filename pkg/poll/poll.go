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

// IsRetryableFunc is the signature for functions that return true if we should
// retry an error
type IsRetryableFunc func(error) bool

// IsAlwaysRetryable instructs WaitWithRetries to retry until time expires.
func IsAlwaysRetryable(error) bool {
	return true
}

// IsNeverRetryable instructs WaitWithRetries not to retry.
func IsNeverRetryable(error) bool {
	return false
}

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

// WaitWithRetries will invoke a function `f` until it returns true or the
// context `ctx` is done. If `f` returns an error, WaitWithRetries will tolerate
// up to `numRetries` errors.
func WaitWithRetries(ctx context.Context, numRetries int, r IsRetryableFunc, f Func) error {
	return WaitWithBackoffWithRetries(ctx, backoff.Backoff{}, numRetries, r, f)
}

// WaitWithBackoffWithRetries will invoke a function `f` until it returns true or the
// context `ctx` is done. If `f` returns an error, WaitWithBackoffWith retries will tolerate
// up to `numRetries` errors. If returned error is not retriable according to `r`, then
// it will bait out immediately. The wait time between retries will be decided by backoff
// parameters `b`.
func WaitWithBackoffWithRetries(ctx context.Context, b backoff.Backoff, numRetries int, r IsRetryableFunc, f Func) error {
	if numRetries < 0 {
		return errors.New("numRetries must be non-negative")
	}

	retries := 0
	for {
		ok, err := f(ctx)
		if err != nil {
			if !r(err) || retries >= numRetries {
				return err
			}
			retries++
		} else if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "Context done while polling")
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
