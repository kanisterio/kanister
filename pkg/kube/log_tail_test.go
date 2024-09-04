// Copyright 2023 The Kanister Authors.
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

//go:build !unit
// +build !unit

package kube

import (
	. "gopkg.in/check.v1"
)

type LogTailTestSuite struct{}

var _ = Suite(&LogTailTestSuite{})

func (s *LogTailTestSuite) TestLogsTail(c *C) {
	for caseIdx, tc := range []struct {
		limit    int
		input    []string
		expected string
	}{
		{2, []string{"line 1", "line 2", "line 3", "line 4", "line 5"}, "line 4\r\nline 5"},
		{2, []string{"line 1\nline 2", "line 3", "line 4\r\nline 5"}, "line 4\r\nline 5"},
		{5, []string{"line 1", "line 2"}, "line 1\r\nline 2"},
		{1, []string{"line 1", "line 2"}, "line 2"},
	} {
		fc := Commentf("Failed for case #%v. Log: %s", caseIdx, tc.expected)
		lt := NewLogTail(tc.limit)

		for _, in := range tc.input {
			w, e := lt.Write([]byte(in))
			c.Check(e, IsNil)
			c.Check(w, Equals, len([]byte(in)))
		}

		r := lt.ToString()
		c.Check(r, Equals, tc.expected, fc)
	}

	lt := NewLogTail(3)
	c.Check(lt.ToString(), Equals, "") // If there were no writes at all, output should be empty line

	_, err := lt.Write([]byte("line1"))
	c.Assert(err, IsNil)
	_, err = lt.Write([]byte("line2"))
	c.Assert(err, IsNil)

	c.Check(lt.ToString(), Equals, "line1\r\nline2")
	c.Check(lt.ToString(), Equals, "line1\r\nline2") // Second invocation should get the same result

	// Check that buffer is still working after ToString is called
	_, err = lt.Write([]byte("line3"))
	c.Assert(err, IsNil)

	c.Check(lt.ToString(), Equals, "line1\r\nline2\r\nline3")

	_, err = lt.Write([]byte("line4"))
	c.Assert(err, IsNil)
	c.Check(lt.ToString(), Equals, "line2\r\nline3\r\nline4")
}
