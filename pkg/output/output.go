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

package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/pkg/errors"
)

const (
	PhaseOpString = "###Phase-output###:"
)

type Output struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func marshalOutput(key, value string) (string, error) {
	out := &Output{
		Key:   key,
		Value: value,
	}
	outString, err := json.Marshal(out)
	if err != nil {
		return "", errors.Wrap(err, "Failed to marshal key-value pair")
	}
	return string(outString), nil
}

// UnmarshalOutput unmarshals output json into Output struct
func UnmarshalOutput(opString []byte) (*Output, error) {
	p := &Output{}
	err := json.Unmarshal([]byte(opString), p)
	return p, errors.Wrap(err, "Failed to unmarshal key-value pair")
}

// ValidateKey validates the key argument
func ValidateKey(key string) error {
	// key should be non-empty
	if key == "" {
		return errors.New("Key should not be empty")
	}
	// key can contain only alpha numeric characters and underscore
	valid := regexp.MustCompile("^[a-zA-Z0-9_]*$").MatchString
	if !valid(key) {
		return errors.New("Key should contain only alphanumeric characters and underscore")
	}
	return nil
}

// PrintOutput runs the `kando output` command
func PrintOutput(key, value string) error {
	return fPrintOutput(os.Stdout, key, value)
}

// PrintOutputTo prints the output of the `kando output` command to w.
func PrintOutputTo(w io.Writer, key, value string) error {
	return fPrintOutput(w, key, value)
}

func fPrintOutput(w io.Writer, key, value string) error {
	outString, err := marshalOutput(key, value)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, PhaseOpString, outString)
	return nil
}
