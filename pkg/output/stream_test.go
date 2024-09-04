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
	"log"
	"math/rand"
	"time"

	. "gopkg.in/check.v1"
	apirand "k8s.io/apimachinery/pkg/util/rand"

	"github.com/kanisterio/kanister/pkg/output"
)

type EndlinePolicy int

const (
	NewlineEndline = '\n'
	NoEndline      = rune(0)

	EndlineRequired EndlinePolicy = iota
	EndlineProhibited
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

		prefixWithEndLine := endlinePolicy == EndlineRequired

		cases[i] = testCase{
			prefixLength:      generateLength(r, avgPrefixLength),
			prefixWithEndline: prefixWithEndLine,
			key:               key,
			value:             value,
		}
	}

	return cases
}

func getTestReaderCloser(done chan struct{}, cases []testCase) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer closePipe(pw)
		writeTestCases(done, pw, cases)
	}()

	return pr
}

func closePipe(pw io.WriteCloser) {
	if err := pw.Close(); err != nil {
		log.Printf("Error %v closing connection", err)
	}
}

func writeTestCases(done chan struct{}, pw io.Writer, cases []testCase) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, tc := range cases {
		if shouldExit(done) {
			return
		}

		if tc.prefixLength != 0 {
			writePrefix(pw, r, tc)
		}

		if tc.key != "" {
			if err := output.PrintOutputTo(pw, tc.key, string(tc.value)); err != nil {
				return
			}
		}
	}
}

func shouldExit(done chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

func writePrefix(pw io.Writer, r *rand.Rand, tc testCase) {
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

// TestLongStreamsWithoutPhaseOutput Will produce 10 long lines
// each line will contain from 50Kb to 60Kb of random text
// there will be no phase output in lines
func (s *OutputTestSuite) TestLongStreamsWithoutPhaseOutput(c *C) {
	done := make(chan struct{})
	defer func() { close(done) }()

	cases := generateTestCases(10, 50000, 0, 0, EndlineRequired)
	r := getTestReaderCloser(done, cases)
	m, e := output.LogAndParse(context.TODO(), r)
	c.Check(e, IsNil)
	c.Check(len(m), Equals, 0)
}

// TestShortStreamsWithPhaseOutput Will produce one short line
// which will contain ONLY phase output and nothing else
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

// TestLongStreamsWithPhaseOutput Will produce 10 long lines
// each line will contain from 10Kb to 12Kb of random text and
// phase output preceded with newline
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

// TestHugeStreamsWithHugePhaseOutput Will produce five huge lines
// each line will contain ±100Kb of random text WITH newline before Phase Output mark
// Phase output value will be very short
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

// TestHugeStreamsWithHugePhaseOutput Will produce five huge lines
// each line will contain ±500Kb of random text WITH newline before Phase Output mark
// Phase output value will be ±10Kb of random text
func (s *OutputTestSuite) TestHugeStreamsWithLongPhaseOutput(c *C) {
	done := make(chan struct{})
	defer func() { close(done) }()

	cases := generateTestCases(5, 500000, 10, 10000, EndlineRequired)
	r := getTestReaderCloser(done, cases)
	m, e := output.LogAndParse(context.TODO(), r)
	c.Check(e, IsNil)
	c.Check(len(m), Equals, 5)
	c.Check(m[cases[0].key], Equals, string(cases[0].value))
}

// TestHugeStreamsWithHugePhaseOutput Will produce one huge line
// which will contain ±500Kb of random text WITH newline before Phase Output mark
// Phase output value will also be ±500Kb
func (s *OutputTestSuite) TestHugeStreamsWithHugePhaseOutput(c *C) {
	done := make(chan struct{})
	defer func() { close(done) }()

	cases := generateTestCases(1, 500000, 10, 500000, EndlineRequired)
	r := getTestReaderCloser(done, cases)
	m, e := output.LogAndParse(context.TODO(), r)
	c.Check(e, IsNil)
	c.Check(len(m), Equals, 1)
	c.Check(m[cases[0].key], Equals, string(cases[0].value))
}

// TestHugeStreamsWithHugePhaseOutputWithoutNewlineDelimiter Will produce one huge line
// which will contain ±500Kb of random text WITHOUT newline before Phase Output mark
// Phase output value will also be ±500Kb
func (s *OutputTestSuite) TestHugeStreamsWithHugePhaseOutputWithoutNewlineDelimiter(c *C) {
	done := make(chan struct{})
	defer func() { close(done) }()

	cases := generateTestCases(1, 500000, 10, 500000, EndlineProhibited)
	r := getTestReaderCloser(done, cases)
	m, e := output.LogAndParse(context.TODO(), r)
	c.Check(e, IsNil)
	c.Check(len(m), Equals, 1)
	c.Check(m[cases[0].key], Equals, string(cases[0].value))
}
