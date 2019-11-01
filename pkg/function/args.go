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

package function

import (
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

// Arg returns the value of the specified argument
// It will return an error if the argument type does not match the result type
func Arg(args map[string]interface{}, argName string, result interface{}) error {
	if val, ok := args[argName]; ok {
		if err := mapstructure.WeakDecode(val, result); err != nil {
			return errors.Wrapf(err, "Failed to decode arg `%s`", argName)
		}
		return nil
	}
	return errors.New("Argument missing " + argName)
}

// OptArg returns the value of the specified argument if it exists
// It will return the default value if the argument does not exist
func OptArg(args map[string]interface{}, argName string, result interface{}, defaultValue interface{}) error {
	if _, ok := args[argName]; ok {
		return Arg(args, argName, result)
	}
	return mapstructure.Decode(defaultValue, result)
}

// ArgExists checks if the argument exists
func ArgExists(args map[string]interface{}, argName string) bool {
	_, ok := args[argName]
	return ok
}

// GetPodSpecOverride merges PodOverride specs passed in args and TemplateParams and returns combined Override specs
func GetPodSpecOverride(tp param.TemplateParams, args map[string]interface{}, argName string) (crv1alpha1.JSONMap, error) {
	var podOverride crv1alpha1.JSONMap
	var err error
	if err = OptArg(args, KubeTaskPodOverrideArg, &podOverride, tp.PodOverride); err != nil {
		return nil, err
	}

	// Check if PodOverride specs are passed through actionset
	// If yes, override podOverride specs
	if tp.PodOverride != nil {
		podOverride, err = kube.CreateAndMergeJsonPatch(podOverride, tp.PodOverride)
		if err != nil {
			return nil, err
		}
	}
	return podOverride, nil
}
