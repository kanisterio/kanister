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

import "fmt"

type indicator string

const (
	Fail indicator = `❌`
	Pass indicator = `✅`
	Skip indicator = `🚫`
)

func PrintStage(description string, i indicator) {
	switch i {
	case Pass:
		fmt.Printf("Passed the '%s' check.. %s\n", description, i)
	case Skip:
		fmt.Printf("Skipping the '%s' check.. %s\n", description, i)
	case Fail:
		fmt.Printf("Failed the '%s' check.. %s\n", description, i)
	default:
		fmt.Println(description)
	}
}
