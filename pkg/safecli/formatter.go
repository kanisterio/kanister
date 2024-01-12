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

package safecli

import "strings"

// ArgumentFormatter defines a function that formats a command argument to the string.
type ArgumentFormatter func(a Argument) string

// format combines arguments using ArgumentFormatter.
func (f ArgumentFormatter) format(args []Argument) []string {
	var c []string
	for _, arg := range args {
		c = append(c, f(arg))
	}
	return c
}

// CommandArgumentFormatter implements regular command formatter.
func CommandArgumentFormatter(a Argument) string {
	return combineKeyValue(a.Key, a.Value.PlainString())
}

// LogArgumentFormatter implements log formatter.
func LogArgumentFormatter(a Argument) string {
	return combineKeyValue(a.Key, a.Value.String())
}

func combineKeyValue(k, v string) string {
	if k == "" {
		return v
	}
	var s strings.Builder
	s.WriteString(k)
	s.WriteString(keyValueDelimiter)
	s.WriteString(v)
	return s.String()
}
