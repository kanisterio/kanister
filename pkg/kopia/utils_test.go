/*
Copyright 2026 by contributors to the Kanister project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kopia_test

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia"
)

func Test(t *testing.T) { check.TestingT(t) }

type UtilsSuite struct{}

var _ = check.Suite(&UtilsSuite{})

func (s *UtilsSuite) TestGetDataStoreGeneralCacheSizes(c *check.C) {
	for _, tc := range []struct {
		opt              map[string]int
		expectedContent  int
		expectedMetadata int
	}{
		{
			opt:              nil,
			expectedContent:  0,
			expectedMetadata: 500,
		},
		{
			opt:              map[string]int{},
			expectedContent:  0,
			expectedMetadata: 500,
		},
		{
			// Each size is read from its own connect option.
			opt: map[string]int{
				kopia.DataStoreGeneralContentCacheSizeMBKey:  1000,
				kopia.DataStoreGeneralMetadataCacheSizeMBKey: 2000,
			},
			expectedContent:  1000,
			expectedMetadata: 2000,
		},
		{
			// The metadata size keeps its default when only the content
			// size is set.
			opt: map[string]int{
				kopia.DataStoreGeneralContentCacheSizeMBKey: 1000,
			},
			expectedContent:  1000,
			expectedMetadata: 500,
		},
		{
			// A zero content size does not disable the metadata cache.
			opt: map[string]int{
				kopia.DataStoreGeneralContentCacheSizeMBKey: 0,
			},
			expectedContent:  0,
			expectedMetadata: 500,
		},
		{
			// The content size keeps its default when only the metadata
			// size is set.
			opt: map[string]int{
				kopia.DataStoreGeneralMetadataCacheSizeMBKey: 2000,
			},
			expectedContent:  0,
			expectedMetadata: 2000,
		},
	} {
		c.Check(kopia.GetDataStoreGeneralContentCacheSize(tc.opt), check.Equals, tc.expectedContent)
		c.Check(kopia.GetDataStoreGeneralMetadataCacheSize(tc.opt), check.Equals, tc.expectedMetadata)
	}
}
