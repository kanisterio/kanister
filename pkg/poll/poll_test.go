// Copyright 2019 Kasten Inc.
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
	"testing"
	"time"

	"github.com/jpillora/backoff"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type PollSuite struct{}

var _ = Suite(&PollSuite{})

type mockPollFunc struct {
	c   *C
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

func (s *PollSuite) TestWaitWithBackoff(c *C) {
	for _, tc := range []struct {
		f       mockPollFunc
		checker Checker
	}{
		{
			f: mockPollFunc{
				c: c,
				res: []pollFuncResult{
					pollFuncResult{ok: true, err: nil},
				},
			},
			checker: IsNil,
		},
		{
			f: mockPollFunc{
				c: c,
				res: []pollFuncResult{
					pollFuncResult{ok: false, err: errFake},
				},
			},
			checker: NotNil,
		},
		{
			f: mockPollFunc{
				c: c,
				res: []pollFuncResult{
					pollFuncResult{ok: true, err: errFake},
				},
			},
			checker: NotNil,
		},
		{
			f: mockPollFunc{
				c: c,
				res: []pollFuncResult{
					pollFuncResult{ok: false, err: nil},
					pollFuncResult{ok: true, err: nil},
				},
			},
			checker: IsNil,
		},
		{
			f: mockPollFunc{
				c: c,
				res: []pollFuncResult{
					pollFuncResult{ok: false, err: nil},
					pollFuncResult{ok: true, err: errFake},
				},
			},
			checker: NotNil,
		},
	} {
		ctx := context.Background()
		b := backoff.Backoff{}
		err := WaitWithBackoff(ctx, b, tc.f.Run)
		c.Check(err, tc.checker)
	}
}

func (s *PollSuite) TestWaitWithBackoffCancellation(c *C) {
	f := func(context.Context) (bool, error) {
		return false, nil
	}
	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Millisecond)
	defer cancel()

	b := backoff.Backoff{}
	err := WaitWithBackoff(ctx, b, f)
	c.Check(err, NotNil)
}

func (s *PollSuite) TestWaitWithBackoffBackoff(c *C) {
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
	c.Assert(err, IsNil)
	c.Assert(time.Now().Sub(now) > (numIterations-1)*time.Millisecond, Equals, true)
}
