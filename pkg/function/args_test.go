// Copyright 2020 The Kanister Authors.
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
	. "gopkg.in/check.v1"
)

var _ = Suite(&ArgsTestSuite{})

type ArgsTestSuite struct {
}

func (s *ArgsTestSuite) TestGetYamlList(c *C) {
	testCases := []struct {
		name       string
		args       map[string]interface{}
		errChecker Checker
		valList    []string
	}{
		{
			name: "Pass key as string in case of configmap reference or artifact reference",
			args: map[string]interface{}{
				"key": "- val1\n- val2\n- val3\n",
			},
			errChecker: IsNil,
			valList:    []string{"val1", "val2", "val3"},
		},
		{
			name: "Pass key as string slice",
			args: map[string]interface{}{
				"key": []string{"test1", "test2", "test3"},
			},
			errChecker: IsNil,
			valList:    []string{"test1", "test2", "test3"},
		},
		{
			name: "Pass key as interface slice",
			args: map[string]interface{}{
				"key": []interface{}{"test1", "test2", "test3"},
			},
			errChecker: IsNil,
			valList:    []string{"test1", "test2", "test3"},
		},
		{
			name: "Invalid key format",
			args: map[string]interface{}{
				"key": "not a slice",
			},
			errChecker: NotNil,
			valList:    nil,
		},
		{
			name: "Nil key arg",
			args: map[string]interface{}{
				"invalid": nil,
			},
			errChecker: IsNil,
			valList:    nil,
		},
	}

	for _, tc := range testCases {
		valList, err := GetYamlList(tc.args, "key")
		c.Check(err, tc.errChecker, Commentf("Test: %s Failed!", tc.name))
		c.Check(valList, DeepEquals, tc.valList, Commentf("Test: %s Failed!", tc.name))
	}
}
