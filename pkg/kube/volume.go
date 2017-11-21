package kube

import (
	"context"
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	pvMatchLabelName = "k10pvmatchid"
	pvcGenerateName  = "kasten-pvc-"
	// NoPVCNameSpecified is used by the caller to indicate that the PVC name
	// should be auto-generated
	NoPVCNameSpecified = ""
)

// CreatePVC creates a PersistentVolumeClaim and returns the name.
// An empty 'targetVolID' indicates the caller would like the PV to be dynamically provisioned
// An empty 'name' indicates the caller would like the name to be auto-generated
func CreatePVC(ctx context.Context, kubeCli kubernetes.Interface, ns string, name string, sizeGB int64, targetVolID string, annotations map[string]string) (string, error) {
	sizeFmt := fmt.Sprintf("%dGi", sizeGB)
	size, err := resource.ParseQuantity(sizeFmt)
	emptyStorageClass := ""
	if err != nil {
		return "", err
	}
	pvc := v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kasten-pvc-",
			Annotations:  annotations,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): size,
				},
			},
		},
	}
	if name != "" {
		pvc.ObjectMeta.Name = name
	} else {
		pvc.ObjectMeta.GenerateName = pvcGenerateName
	}

	// Check if a targetVolID is empty i.e. dynamic provisioning is desired

	if targetVolID != "" {
		pvc.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{pvMatchLabelName: targetVolID},
		}
		// Spec.StorageClassName = "" for disable dynamic provisioning
		pvc.Spec.StorageClassName = &emptyStorageClass
	}
	createdPVC, err := kubeCli.CoreV1().PersistentVolumeClaims(ns).Create(&pvc)
	if err != nil {
		return "", err
	}
	return createdPVC.Name, nil
}
