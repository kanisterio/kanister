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

type SnapshotManager struct {
	config  map[string]string
	ivdPETM *ivd.IVDProtectedEntityTypeManager
}

func NewSnapshotManager(config map[string]string) (*SnapshotManager, error) {
	ep, ok := config[kvm.VSphereEndpointKey]
	if !ok {
		return nil, fmt.Errorf("missing endpoint value")
	}
	username, ok := config[kvm.VSphereUsernameKey]
	if !ok {
		return nil, fmt.Errorf("missing username value")
	}
	password, ok := config[kvm.VSpherePasswordKey]
	if !ok {
		return nil, fmt.Errorf("missing password value")
	}
	s3URLBase, ok := config["s3URLBase"]
	if !ok {
		return nil, fmt.Errorf("missing s3URLBase value")
	}
	params := map[string]interface{}{
		"vcHost":     ep,
		"vcUser":     username,
		"vcPassword": password,
	}
	ivdPETM, err := ivd.NewIVDProtectedEntityTypeManagerFromConfig(params, s3URLBase, logrus.New())
	if err != nil {
		return nil, fmt.Errorf("Unable to create ivd Protected Entity Manager from config %s", err.Error())
	}
	return &SnapshotManager{
		config:  config,
		ivdPETM: ivdPETM,
	}, nil
}

// func (s *SnapshotManager) CreateVolumeFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot) (astrolabe.ProtectedEntityID, error) {
// 	provider, err := kvm.NewProvider(s.config)
// 	if err != nil {
// 		return nil, err
// 	}
// 	tags := map[string]string{}
// 	vol, err := provider.VolumeCreateFromSnapshot(ctx, snapshot, tags)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return astrolabe.NewProtectedEntityIDFromString(vol.ID)
// }
