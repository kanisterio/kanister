// Copyright 2024 The Kanister Authors.
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

package output_test

import (
	"context"
	"io"
	"math/rand"
	"time"

	"github.com/kanisterio/kanister/pkg/output"
	. "gopkg.in/check.v1"
	apirand "k8s.io/apimachinery/pkg/util/rand"
)

type EndlinePolicy int

const (
	NewlineEndline = '\n'
	NoEndline      = rune(0)

	EndlineRequired EndlinePolicy = iota
	EndlineRandom
)

type OutputTestSuite struct{}

var _ = Suite(&OutputTestSuite{})

type testCase struct {
	prefixLength      int
	prefixWithEndline bool
	key               string
	value             []rune
}

func generateLength(r *rand.Rand, avgLength int) int {
	if avgLength == 0 {
		return 0
	}
	return r.Intn(avgLength/5) + avgLength // Return random length ±20% of avgLength
}

var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_!=-+*/\\")

func generateRandomRunes(r *rand.Rand, length int, endline rune) []rune {
	totalLength := length
	if endline != NoEndline {
		totalLength += 1
	}
	line := make([]rune, totalLength)
	var last rune
	for j := 0; j < length; j++ {
		var current rune
		for rpt := true; rpt; rpt = last == '\\' && (current == '\n' || current == '\r') {
			current = runes[r.Intn(len(runes))]
		}

		line[j] = current
		last = current
	}

	if endline != NoEndline {
		line[length] = endline
	}
	return line
}

func generateTestCases(numOfLines, avgPrefixLength, avgKeyLength, avgValueLength int, endlinePolicy EndlinePolicy) []testCase {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	cases := make([]testCase, numOfLines)
	for i := 0; i < numOfLines; i++ {
		key := ""
		value := []rune{}
		if avgKeyLength != 0 {
			key = apirand.String(generateLength(r, avgKeyLength))
			value = generateRandomRunes(r, avgValueLength, NoEndline)
		}
		cases[i] = testCase{
			prefixLength:      generateLength(r, avgPrefixLength),
			prefixWithEndline: endlinePolicy == EndlineRequired || rand.Intn(2) == 1,
			key:               key,
			value:             value,
		}
	}

	return cases
}

func getTestReaderCloser(done chan struct{}, cases []testCase) io.ReadCloser {
	pr, pw := io.Pipe()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	go func() {
		defer pw.Close()

		for _, tc := range cases {
			select {
			case <-done:
				return
			default:
				if tc.prefixLength != 0 {
					endline := NoEndline
					if tc.prefixWithEndline {
						endline = NewlineEndline
					}
					prefixLine := generateRandomRunes(r, tc.prefixLength, endline)
					_, err := pw.Write([]byte(string(prefixLine)))
					if err != nil {
						return
					}
				}

				if tc.key != "" {
					err := output.PrintOutputTo(pw, tc.key, string(tc.value))
					if err != nil {
						return
					}
				}
			}
		}
	}()

	return pr
}

func (s *OutputTestSuite) TestHugeStreamsWithoutPhaseOutput(c *C) {
	done := make(chan struct{})
	defer func() { close(done) }()

	// e-sumin: Here I'm generating test case, when we have just random string around 10000 runes
	// I expect that it will be logged as one line as it was before
	// But in fact it is logged by chunks of ±4kb
	// When we will fix the code behavior, numOfLines has to be set to 10, and avgPrefix len has to be set to 500000
	cases := generateTestCases(10, 50000, 0, 0, EndlineRequired)
	r := getTestReaderCloser(done, cases)
	m, e := output.LogAndParse(context.TODO(), r)
	c.Check(e, IsNil)
	c.Check(len(m), Equals, 0)
}

func (s *OutputTestSuite) TestShortStreamsWithPhaseOutput(c *C) {
	done := make(chan struct{})
	defer func() { close(done) }()

	cases := generateTestCases(1, 0, 10, 50, EndlineRequired)
	r := getTestReaderCloser(done, cases)
	m, e := output.LogAndParse(context.TODO(), r)
	c.Check(e, IsNil)
	c.Check(len(m), Equals, 1)
	c.Check(m[cases[0].key], Equals, string(cases[0].value))
}

func (s *OutputTestSuite) TestLongStreamsWithPhaseOutput(c *C) {
	done := make(chan struct{})
	defer func() { close(done) }()

	cases := generateTestCases(10, 10000, 10, 50, EndlineRequired)
	r := getTestReaderCloser(done, cases)
	m, e := output.LogAndParse(context.TODO(), r)
	c.Check(e, IsNil)
	c.Check(len(m), Equals, 10)
	c.Check(m[cases[0].key], Equals, string(cases[0].value))
}

func (s *OutputTestSuite) TestHugeStreamsWithPhaseOutput(c *C) {
	done := make(chan struct{})
	defer func() { close(done) }()

	cases := generateTestCases(5, 100000, 10, 50, EndlineRequired)
	r := getTestReaderCloser(done, cases)
	m, e := output.LogAndParse(context.TODO(), r)
	c.Check(e, IsNil)
	c.Check(len(m), Equals, 5)
	c.Check(m[cases[0].key], Equals, string(cases[0].value))
}
