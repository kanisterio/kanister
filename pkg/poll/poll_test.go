// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package poll

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/jpillora/backoff"
	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type PollSuite struct{}

var _ = check.Suite(&PollSuite{})

type mockPollFunc struct {
	c   *check.C
	res []pollFuncResult
}

type pollFuncResult struct {
	ok  bool
	err error
}

func (mpf *mockPollFunc) Run(ctx context.Context) (bool, error) {
	if mpf == nil || len(mpf.res) == 0 {
		mpf.c.FailNow()
	}
	ok, err := mpf.res[0].ok, mpf.res[0].err
	if len(mpf.res) == 1 {
		mpf.res = nil
	} else {
		mpf.res = mpf.res[1:]
	}
	return ok, err
}

var errFake = fmt.Errorf("THIS IS FAKE")

func (s *PollSuite) TestWaitWithBackoff(c *check.C) {
	for _, tc := range []struct {
		f       mockPollFunc
		checker check.Checker
	}{
		{
			f: mockPollFunc{
				c: c,
				res: []pollFuncResult{
					{ok: true, err: nil},
				},
			},
			checker: check.IsNil,
		},
		{
			f: mockPollFunc{
				c: c,
				res: []pollFuncResult{
					{ok: false, err: errFake},
				},
			},
			checker: check.NotNil,
		},
		{
			f: mockPollFunc{
				c: c,
				res: []pollFuncResult{
					{ok: true, err: errFake},
				},
			},
			checker: check.NotNil,
		},
		{
			f: mockPollFunc{
				c: c,
				res: []pollFuncResult{
					{ok: false, err: nil},
					{ok: true, err: nil},
				},
			},
			checker: check.IsNil,
		},
		{
			f: mockPollFunc{
				c: c,
				res: []pollFuncResult{
					{ok: false, err: nil},
					{ok: true, err: errFake},
				},
			},
			checker: check.NotNil,
		},
	} {
		ctx := context.Background()
		b := backoff.Backoff{}
		err := WaitWithBackoff(ctx, b, tc.f.Run)
		c.Check(err, tc.checker)
	}
}

func (s *PollSuite) TestWaitWithBackoffCancellation(c *check.C) {
	f := func(context.Context) (bool, error) {
		return false, nil
	}
	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Millisecond)
	defer cancel()

	b := backoff.Backoff{}
	err := WaitWithBackoff(ctx, b, f)
	c.Check(err, check.NotNil)
}

func (s *PollSuite) TestWaitWithRetriesTimeout(c *check.C) {
	// There's a better chance of catching a race condition
	// if there is only one thread
	maxprocs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(maxprocs)

	f := func(context.Context) (bool, error) {
		return false, errkit.New("retryable")
	}
	errf := func(err error) bool {
		return err.Error() == "retryable"
	}
	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Millisecond)
	defer cancel()

	backoff := backoff.Backoff{}
	backoff.Min = 2 * time.Millisecond
	err := WaitWithBackoffWithRetries(ctx, backoff, 10, errf, f)
	c.Check(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*context deadline exceeded*")
}

func (s *PollSuite) TestWaitWithBackoffBackoff(c *check.C) {
	const numIterations = 10
	i := 0
	f := func(context.Context) (bool, error) {
		i++
		if i < numIterations {
			return false, nil
		}
		return true, nil
	}
	ctx := context.Background()
	b := backoff.Backoff{
		Min: time.Millisecond,
		Max: time.Millisecond,
	}

	now := time.Now()
	err := WaitWithBackoff(ctx, b, f)
	c.Assert(err, check.IsNil)
	c.Assert(time.Since(now) > (numIterations-1)*time.Millisecond, check.Equals, true)
}
