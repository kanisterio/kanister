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
	"context"
	"flag"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/astrolabe/pkg/astrolabe"
	"github.com/vmware-tanzu/astrolabe/pkg/ivd"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	kvm "github.com/kanisterio/kanister/pkg/blockstorage/vmware"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

func main() {
	ctx := context.Background()

	snapshot := flag.String("snapid", "", "snapshot id of the form type:volumeID:snapshotID")
	vcHost := flag.String("vchost", "", "vSphere endpoint")
	vcUser := flag.String("vcuser", "", "vSphere username")
	vcPass := flag.String("vcpass", "", "vSphere password")
	s3ID := flag.String("s3id", "", "s3 Access ID")
	s3Secret := flag.String("s3secret", "", "s3 Secret")
	s3Bucket := flag.String("s3bucket", "", "s3 Bucket")
	s3Endpoint := flag.String("s3ep", "", "s3 Endpoint")
	s3Region := flag.String("s3region", "", "s3 Region")
	s3Prefix := flag.String("s3prefix", "", "s3 Prefix")

	flag.Parse()
	if *snapshot == "" {
		fmt.Printf("Error: snapshot id required")
		return
	}

	config := map[string]string{
		kvm.VSphereEndpointKey: *vcHost,
		kvm.VSphereUsernameKey: *vcUser,
		kvm.VSpherePasswordKey: *vcPass,
		"s3URLBase":            "justToPlacateIVD",
	}

	profile := param.Profile{
		Location: crv1alpha1.Location{
			Type:     crv1alpha1.LocationTypeS3Compliant,
			Bucket:   *s3Bucket,
			Endpoint: *s3Endpoint,
			Prefix:   *s3Prefix,
			Region:   *s3Region,
		},
		Credential: param.Credential{
			Type: param.CredentialTypeKeyPair,
			KeyPair: &param.KeyPair{
				ID:     *s3ID,
				Secret: *s3Secret,
			},
		},
		SkipSSLVerify: false,
	}
	// expecting a snapshot id of the form type:volumeID:snapshotID
	err := copySnapshotToObjectStore(ctx, config, profile, *snapshot)
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
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

func copySnapshotToObjectStore(ctx context.Context, config map[string]string, profile param.Profile, snapshot string) error {
	fmt.Println(snapshot)
	snapManager, err := NewSnapshotManager(config)
	if err != nil {
		return err
	}

	// expecting a snapshot id of the form type:volumeID:snapshotID
	peID, err := astrolabe.NewProtectedEntityIDFromString(snapshot)
	if err != nil {
		return err
	}
	pe, err := snapManager.ivdPETM.GetProtectedEntity(ctx, peID)
	if err != nil {
		return err
	}

	reader, err := pe.GetDataReader(ctx)
	if err != nil {
		return err
	}
	path := ""
	return location.Write(ctx, reader, profile, path)
}
