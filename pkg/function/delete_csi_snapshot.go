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

package function

import (
	"context"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	_ = kanister.Register(&deleteCSISnapshotFunc{})
}

var (
	_ kanister.Func = (*deleteCSISnapshotFunc)(nil)
)

const (
	// DeleteCSIVolumeSnapshotFuncName gives the name of the function
	DeleteCSISnapshotFuncName = "DeleteCSISnapshot"
	// DeleteCSISnapshotNameArg provides name of the VolumeSnapshot that needs to be deleted
	DeleteCSISnapshotNameArg = "name"
	// DeleteCSISnapshotNamespaceArg mentions the namespace where the VolumeSnapshot resides
	DeleteCSISnapshotNamespaceArg = "namespace"
)

type deleteCSISnapshotFunc struct{}

func (*deleteCSISnapshotFunc) Name() string {
	return DeleteCSISnapshotFuncName
}

func (*deleteCSISnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var name, namespace string
	if err := Arg(args, DeleteCSISnapshotNameArg, &name); err != nil {
		return nil, err
	}
	if err := Arg(args, DeleteCSISnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	kubeCli, err := kube.NewClient()
	if err != nil {
		return nil, err
	}
	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return nil, err
	}
	snapshotter, err := snapshot.NewSnapshotter(kubeCli, dynCli)
	if err != nil {
		return nil, err
	}
	if _, err := snapshotter.Delete(ctx, name, namespace); err != nil {
		return nil, err
	}
	return nil, nil
}

func (*deleteCSISnapshotFunc) RequiredArgs() []string {
	return []string{
		DeleteCSISnapshotNameArg,
		DeleteCSISnapshotNamespaceArg,
	}
}
