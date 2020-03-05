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

package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/astrolabe/pkg/ivd"

	kvm "github.com/kanisterio/kanister/pkg/blockstorage/vmware"
)

func main() {
	config := map[string]string{
		kvm.VSphereEndpointKey: "hostname",
		kvm.VSphereUsernameKey: "user",
		kvm.VSpherePasswordKey: "password",
		"s3URLBase":            "justToPlacateIVD",
	}
	fmt.Printf("Error: %v", throwAwayJustForLinking(config))
}

// Based on code in:
// - kanister/pkg/blockstorage/vmware
// - vmware/astrolabe/pkg/ivd
func throwAwayJustForLinking(config map[string]string) error {
	ep, ok := config[kvm.VSphereEndpointKey]
	if !ok {
		return fmt.Errorf("missing endpoint value")
	}
	username, ok := config[kvm.VSphereUsernameKey]
	if !ok {
		return fmt.Errorf("missing username value")
	}
	password, ok := config[kvm.VSpherePasswordKey]
	if !ok {
		return fmt.Errorf("missing password value")
	}
	s3URLBase, ok := config["s3URLBase"]
	if !ok {
		return fmt.Errorf("missing s3URLBase value")
	}

	params := map[string]interface{}{
		"vcHost":     ep,
		"vcUser":     username,
		"vcPassword": password,
	}
	_, err := ivd.NewIVDProtectedEntityTypeManagerFromConfig(params, s3URLBase, logrus.New())
	return err
}
