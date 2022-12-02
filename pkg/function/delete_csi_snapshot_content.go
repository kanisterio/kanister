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
	_ = kanister.Register(&deleteCSISnapshotContentFunc{NewBaseFunction()})
}

var (
	_ kanister.Func = (*deleteCSISnapshotContentFunc)(nil)
)

const (
	// DeleteCSIVolumeSnapshotFuncName gives the name of the function
	DeleteCSISnapshotContentFuncName = "DeleteCSISnapshotContent"

	// DeleteCSISnapshotContentNameArg provides name of the VolumeSnapshotContent that needs to be deleted
	DeleteCSISnapshotContentNameArg = "name"
)

type deleteCSISnapshotContentFunc struct{ *BaseFunction }

func (*deleteCSISnapshotContentFunc) Name() string {
	return DeleteCSISnapshotContentFuncName
}

func (*deleteCSISnapshotContentFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var name string
	if err := Arg(args, DeleteCSISnapshotContentNameArg, &name); err != nil {
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

	return nil, deleteCSISnapshotContent(ctx, snapshotter, name)
}

func (*deleteCSISnapshotContentFunc) RequiredArgs() []string {
	return []string{
		DeleteCSISnapshotContentNameArg,
	}
}

func (*deleteCSISnapshotContentFunc) Arguments() []string {
	return []string{
		DeleteCSISnapshotContentNameArg,
	}
}

func deleteCSISnapshotContent(ctx context.Context, snapshotter snapshot.Snapshotter, name string) error {
	return snapshotter.DeleteContent(ctx, name)
}
