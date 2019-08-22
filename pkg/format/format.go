// Copyright 2019 Kasten Inc.
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

package format

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

func Log(podName string, containerName string, output string) {
	if output != "" {
		logs := regexp.MustCompile("[\r\n]").Split(output, -1)
		for _, l := range logs {
			if strings.TrimSpace(l) != "" {
				log.Info("Pod: ", podName, " Container: ", containerName, " Out: ", l)
			}
		}
	}
}
