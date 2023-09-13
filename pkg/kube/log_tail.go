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

package kube

import (
	"bufio"
	"bytes"
	"strings"
)

const logTailDefaultLength = 10

// LogTail interface allows to store last N lines of log written to it
type LogTail interface {
	Write(p []byte) (int, error)
	ToString() string
}

type logTail struct {
	lines []string
	idx   int
	len   int
}

// NewLogTail creates logTail struct containing circular buffer for storing `len` last lines of log written through Write method
func NewLogTail(len int) LogTail {
	return &logTail{
		lines: make([]string, len),
		len:   len,
	}
}

// Write implements io.Writer interface. It writes log line(s) to circular buffer
func (lt *logTail) Write(p []byte) (int, error) {
	s := bufio.NewScanner(bytes.NewReader(p))
	for s.Scan() { // Scan log lines one by one.
		l := s.Text()
		l = strings.TrimSpace(l)
		if l == "" { // Skip empty lines since we are not interested in them
			continue
		}
		lt.lines[lt.idx%lt.len] = l
		lt.idx += 1
	}

	return len(p), nil
}

// ToString returns collected lines joined with a newline
func (lt *logTail) ToString() string {
	var result string

	min := 0
	if lt.idx > lt.len {
		min = lt.idx - lt.len
	}

	for i := min; i < lt.idx; i++ {
		line := lt.lines[i%lt.len]
		result += line
		if i != lt.idx-1 {
			result += "\r\n"
		}
	}

	return result
}
