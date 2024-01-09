package format

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/output"
)

func Test(t *testing.T) { TestingT(t) }

type FormatTest struct{}

var _ = Suite(&FormatTest{})

func (s *FormatTest) TestLogToForPhaseOutputs(c *C) {
	const (
		pod       = "test-pod-logto"
		container = "test-container-logto"
	)

	// invariant: format.LogTo() will not format phase outputs.

	var testCases = []struct {
		keys   []string
		values []string
	}{
		{
			keys:   []string{"key1"},
			values: []string{"value1"},
		},
		{
			keys:   []string{"key1", "key2"},
			values: []string{"value1", "value2"},
		},
		{
			keys:   []string{"key1", "key2", "key3"},
			values: []string{"value1", "value2", "value3"},
		},
	}

	for _, tc := range testCases {
		expected := ""
		input := &bytes.Buffer{}
		actual := &bytes.Buffer{}

		for i, key := range tc.keys {
			// create the phase output for each pair of the given k/v
			kv := &bytes.Buffer{}
			err := output.PrintOutputTo(kv, key, tc.values[i])
			c.Assert(err, IsNil)

			kvRaw := fmt.Sprintf("%s\n", kv.String())
			if _, err := input.WriteString(kvRaw); err != nil {
				c.Assert(err, IsNil)
			}

			expected += fmt.Sprintf("%s {\"key\":\"%s\",\"value\":\"%s\"}\n", output.PhaseOpString, key, tc.values[i])
		}
		LogTo(actual, pod, container, input.String())
		c.Check(expected, DeepEquals, actual.String())
	}
}

func (s *FormatTest) TestLogToForNormalLogs(c *C) {
	const (
		pod       = "test-pod-logto"
		container = "test-container-logto"
	)

	var testCases = []struct {
		input    string
		expected string
		count    int // count represents how many "Out"s in the results
	}{
		{
			input:    "",
			expected: "",
			count:    1,
		},
		{
			input:    "test logs",
			expected: `"Out":"test logs"`,
			count:    1,
		},
		{
			input:    "test logs\ntest logs",
			expected: `"Out":"test logs"`,
			count:    2, // the line break causes 2 log lines to be printed
		},
	}

	for _, tc := range testCases {
		actual := &bytes.Buffer{}
		LogTo(actual, pod, container, tc.input)

		c.Assert(strings.Contains(actual.String(), tc.expected), Equals, true)
		c.Assert(strings.Count(actual.String(), tc.expected), Equals, tc.count)
	}
}
