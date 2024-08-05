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
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

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

// GetNamespaceUID gets the UID of the given namespace
func GetNamespaceUID(ctx context.Context, cli kubernetes.Interface, namespace string) (string, error) {
	ns, err := cli.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get namespace %s", namespace)
	}
	return string(ns.GetUID()), nil
}

func GetEnvAsIntOrDefault(envKey string, def int) int {
	if v, ok := os.LookupEnv(envKey); ok {
		iv, err := strconv.Atoi(v)
		if err == nil {
			return iv
		}
		log.WithError(err).Print("Conversion to integer failed. Using default value", field.M{envKey: v, "default_value": def})
	}

	return def
}

func GetEnvAsStringOrDefault(envKey string, def string) string {
	if v, ok := os.LookupEnv(envKey); ok {
		return v
	}

	return def
}

func GetIntOrDefault(value string, defaultValue int) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil {
		v = defaultValue
		return v, errors.New("conversion to integer failed, using default value for the field")
	}
	return v, nil
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

// CheckRequiredArgs checks if the required args of a function `reqArgs` are
// provided in created blueprint's functions' args (`args`).
func CheckRequiredArgs(reqArgs []string, args map[string]interface{}) error {
	for _, a := range reqArgs {
		if _, ok := args[a]; !ok {
			return errors.Errorf("Required arg missing: %s", a)
		}
	}
	return nil
}

// CheckSupportedArgs checks that all the provided (using blueprint) args of a function
// are supported by the kanister function.
func CheckSupportedArgs(supportedArgs []string, args map[string]interface{}) error {
	for a := range args {
		if !slices.Contains(supportedArgs, a) {
			return errors.Errorf("argument %s is not supported", a)
		}
	}
	return nil
}
