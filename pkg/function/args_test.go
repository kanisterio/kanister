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

func (s *ArgsTestSuite) TestGetSecurityGroups(c *C) {
	testCases := []struct {
		name       string
		args       map[string]interface{}
		errChecker Checker
		sgIDs      []string
	}{
		{
			name: "Pass securityGroupID as string",
			args: map[string]interface{}{
				"securityGroupID": "[\"sgID1\", \"sgID2\"]",
			},
			errChecker: IsNil,
			sgIDs:      []string{"sgID1", "sgID2"},
		},
		{
			name: "Pass securityGroupID as string slice",
			args: map[string]interface{}{
				"securityGroupID": []string{"sgIDA", "sgIDB", "SGIDC"},
			},
			errChecker: IsNil,
			sgIDs:      []string{"sgIDA", "sgIDB", "SGIDC"},
		},
		{
			name: "Pass securityGroupID interface slice",
			args: map[string]interface{}{
				"securityGroupID": []interface{}{"sgIDA", "sgIDB", "SGIDC"},
			},
			errChecker: IsNil,
			sgIDs:      []string{"sgIDA", "sgIDB", "SGIDC"},
		},
		{
			name: "Invalid securityGroupID format",
			args: map[string]interface{}{
				"securityGroupID": "not a slice",
			},
			errChecker: NotNil,
			sgIDs:      nil,
		},
		{
			name: "Nil securityGroupID arg",
			args: map[string]interface{}{
				"invalid": nil,
			},
			errChecker: IsNil,
			sgIDs:      nil,
		},
	}

	for _, tc := range testCases {
		sgIDs, err := GetSecurityGroups(tc.args, "securityGroupID")
		c.Check(err, tc.errChecker, Commentf("Test: %s Failed!", tc.name))
		c.Check(sgIDs, DeepEquals, tc.sgIDs, Commentf("Test: %s Failed!", tc.name))
	}
}

func (s *ArgsTestSuite) TestGetDatabases(c *C) {
	testCases := []struct {
		name       string
		args       map[string]interface{}
		errChecker Checker
		dbList     []string
	}{
		{
			name: "Pass databases as string by configmap reference",
			args: map[string]interface{}{
				"databases": "- db1\n- db2\n- db3\n",
			},
			errChecker: IsNil,
			dbList:     []string{"db1", "db2", "db3"},
		},
		{
			name: "Pass databases as string slice",
			args: map[string]interface{}{
				"databases": []string{"test1", "test2", "test3"},
			},
			errChecker: IsNil,
			dbList:     []string{"test1", "test2", "test3"},
		},
		{
			name: "Pass databases as interface slice",
			args: map[string]interface{}{
				"databases": []interface{}{"test1", "test2", "test3"},
			},
			errChecker: IsNil,
			dbList:     []string{"test1", "test2", "test3"},
		},
		{
			name: "Invalid databases format",
			args: map[string]interface{}{
				"databases": "not a slice",
			},
			errChecker: NotNil,
			dbList:     nil,
		},
		{
			name: "Nil databases arg",
			args: map[string]interface{}{
				"invalid": nil,
			},
			errChecker: IsNil,
			dbList:     nil,
		},
	}

	for _, tc := range testCases {
		dbList, err := GetDatabases(tc.args, "databases")
		c.Check(err, tc.errChecker, Commentf("Test: %s Failed!", tc.name))
		c.Check(dbList, DeepEquals, tc.dbList, Commentf("Test: %s Failed!", tc.name))
	}
}
