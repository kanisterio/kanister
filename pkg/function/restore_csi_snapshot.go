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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	_ = kanister.Register(&restoreCSISnapshotFunc{})
}

var (
	_ kanister.Func = (*restoreCSISnapshotFunc)(nil)
)

const (
	// Snapshot API Group
	SnapshotAPIGroup = "snapshot.storage.k8s.io"
	// RestoreCSISnapshotFuncName gives the name of the function
	RestoreCSISnapshotFuncName = "RestoreCSISnapshot"
	// RestoreCSISnapshotNameArg provides name of the VolumeSnapshot
	RestoreCSISnapshotNameArg = "name"
	// RestoreCSISnapshotPVCNameArg gives the name of the newly restored PVC
	RestoreCSISnapshotPVCNameArg = "pvc"
	// RestoreCSISnapshotNamespaceArg mentions the namespace of the newly restored PVC
	RestoreCSISnapshotNamespaceArg = "namespace"
	// RestoreCSISnapshotStorageClassArg specifies the name of the StorageClass
	RestoreCSISnapshotStorageClassArg = "storageClass"
	// RestoreCSISnapshotStorageRequestArg provides the storage size to be requested for PV in PVC
	RestoreCSISnapshotRestoreSizeArg = "restoreSize"
	// RestoreCSISnapshotAccessModesArg lists down the accessmodes for the underlying PV
	RestoreCSISnapshotAccessModesArg = "accessModes"
	// RestoreCSISnapshotLabelsArg has labels that will be added to the newly restored PVC
	RestoreCSISnapshotLabelsArg = "labels"
	// RestoreCSISnapshotVolumeModeArg defines mode of volume
	RestoreCSISnapshotVolumeModeArg = "volumeMode"
)

type restoreCSISnapshotFunc struct{}

type restoreCSISnapshotArgs struct {
	Name         string
	PVC          string
	Namespace    string
	StorageClass string
	RestoreSize  *resource.Quantity
	AccessModes  []corev1.PersistentVolumeAccessMode
	Labels       map[string]string
	VolumeMode   corev1.PersistentVolumeMode
}

func (*restoreCSISnapshotFunc) Name() string {
	return RestoreCSISnapshotFuncName
}

func (*restoreCSISnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var restoreSize string
	var restoreArgs restoreCSISnapshotArgs
	if err := Arg(args, RestoreCSISnapshotNameArg, &restoreArgs.Name); err != nil {
		return nil, err
	}
	if err := Arg(args, RestoreCSISnapshotPVCNameArg, &restoreArgs.PVC); err != nil {
		return nil, err
	}
	if err := Arg(args, RestoreCSISnapshotNamespaceArg, &restoreArgs.Namespace); err != nil {
		return nil, err
	}
	if err := Arg(args, RestoreCSISnapshotStorageClassArg, &restoreArgs.StorageClass); err != nil {
		return nil, err
	}
	if err := Arg(args, RestoreCSISnapshotRestoreSizeArg, &restoreSize); err != nil {
		return nil, err
	}
	if err := OptArg(args, RestoreCSISnapshotAccessModesArg, &restoreArgs.AccessModes, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}); err != nil {
		return nil, err
	}
	if err := validateVolumeAccessModesArg(restoreArgs.AccessModes); err != nil {
		return nil, err
	}
	if err := OptArg(args, RestoreCSISnapshotVolumeModeArg, &restoreArgs.VolumeMode, corev1.PersistentVolumeFilesystem); err != nil {
		return nil, err
	}
	if err := validateVolumeModeArg(restoreArgs.VolumeMode); err != nil {
		return nil, err
	}
	if err := OptArg(args, RestoreCSISnapshotLabelsArg, &restoreArgs.Labels, nil); err != nil {
		return nil, err
	}
	size, err := resource.ParseQuantity(restoreSize)
	if err != nil {
		return nil, err
	}
	if size.IsZero() {
		return nil, fmt.Errorf("Failed to restore CSI snapshot. restoreSize argument cannot be zero")
	}
	restoreArgs.RestoreSize = &size

	kubeCli, err := getClient()
	if err != nil {
		return nil, err
	}
	if _, err := restoreCSISnapshot(ctx, kubeCli, restoreArgs); err != nil {
		return nil, err
	}
	return nil, nil
}

func (*restoreCSISnapshotFunc) RequiredArgs() []string {
	return []string{
		RestoreCSISnapshotNameArg,
		RestoreCSISnapshotPVCNameArg,
		RestoreCSISnapshotNamespaceArg,
		RestoreCSISnapshotStorageClassArg,
		RestoreCSISnapshotRestoreSizeArg,
	}
}

func (*restoreCSISnapshotFunc) Arguments() []string {
	return []string{
		RestoreCSISnapshotNameArg,
		RestoreCSISnapshotPVCNameArg,
		RestoreCSISnapshotNamespaceArg,
		RestoreCSISnapshotStorageClassArg,
		RestoreCSISnapshotRestoreSizeArg,
		RestoreCSISnapshotAccessModesArg,
		RestoreCSISnapshotVolumeModeArg,
		RestoreCSISnapshotLabelsArg,
	}
}

func getClient() (kubernetes.Interface, error) {
	kubeCli, err := kube.NewClient()
	return kubeCli, err
}

func restoreCSISnapshot(ctx context.Context, kubeCli kubernetes.Interface, args restoreCSISnapshotArgs) (*corev1.PersistentVolumeClaim, error) {
	pvc := newPVCManifest(args)
	if _, err := kubeCli.CoreV1().PersistentVolumeClaims(args.Namespace).Create(ctx, pvc, metav1.CreateOptions{}); err != nil {
		return nil, err
	}
	return pvc, nil
}

func newPVCManifest(args restoreCSISnapshotArgs) *corev1.PersistentVolumeClaim {
	snapshotAPIGroup := SnapshotAPIGroup
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      args.PVC,
			Namespace: args.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: args.AccessModes,
			VolumeMode:  &args.VolumeMode,
			DataSource: &corev1.TypedLocalObjectReference{
				APIGroup: &snapshotAPIGroup,
				Kind:     "VolumeSnapshot",
				Name:     args.Name,
			},
			StorageClassName: &args.StorageClass,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *args.RestoreSize,
				},
			},
		},
	}
	if args.Labels != nil {
		pvc.ObjectMeta.Labels = args.Labels
	}
	return pvc
}

func validateVolumeModeArg(volumeMode corev1.PersistentVolumeMode) error {
	switch volumeMode {
	case corev1.PersistentVolumeFilesystem,
		corev1.PersistentVolumeBlock:
	default:
		return errors.New("Given volumeMode " + string(volumeMode) + " is invalid")
	}
	return nil
}

func validateVolumeAccessModesArg(accessModes []corev1.PersistentVolumeAccessMode) error {
	for _, accessModeInArg := range accessModes {
		switch accessModeInArg {
		case corev1.ReadOnlyMany,
			corev1.ReadWriteMany,
			corev1.ReadWriteOnce:
		default:
			return errors.New("Given accessMode " + string(accessModeInArg) + " is invalid")
		}
	}
	return nil
}
