// Copyright 2022 The Kanister Authors.
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

package utils

import (
	"strings"
	"time"
)

// RoundUpDuration rounds duration to highest set duration unit
func RoundUpDuration(t time.Duration) time.Duration {
	if t < time.Minute {
		return t.Round(time.Second)
	}
	if t < time.Hour {
		return t.Round(time.Minute)
	}
	return t.Round(time.Hour)
}

// DurationToString formats the given duration into a short format which eludes trailing zero units in the string.
func DurationToString(d time.Duration) string {
	s := d.String()

	if strings.HasSuffix(s, "h0m0s") {
		return s[:len(s)-4]
	}

	if strings.HasSuffix(s, "m0s") {
		return s[:len(s)-2]
	}

	return s
}
