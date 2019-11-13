package function

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

const (
	FunctionOutputVersion = "version"
)

// ValidateCredentials verifies if the given credentials have appropriate values set
func ValidateCredentials(creds *param.Credential) error {
	if creds == nil {
		return errors.New("Empty credentials")
	}
	switch creds.Type {
	case param.CredentialTypeKeyPair:
		if creds.KeyPair == nil {
			return errors.New("Empty KeyPair field")
		}
		if len(creds.KeyPair.ID) == 0 {
			return errors.New("Access key ID is not set")
		}
		if len(creds.KeyPair.Secret) == 0 {
			return errors.New("Secret access key is not set")
		}
		return nil
	case param.CredentialTypeSecret:
		return secrets.ValidateCredentials(creds.Secret)
	default:
		return errors.Errorf("Unsupported type '%s' for credentials", creds.Type)
	}
}

// ValidateProfile verifies if the given profile has valid creds and location type
func ValidateProfile(profile *param.Profile) error {
	if profile == nil {
		return errors.New("Profile must be non-nil")
	}
	if err := ValidateCredentials(&profile.Credential); err != nil {
		return err
	}
	switch profile.Location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
	case crv1alpha1.LocationTypeGCS:
	case crv1alpha1.LocationTypeAzure:
	default:
		return errors.New("Location type not supported")
	}
	return nil
}

// GetPodWriter creates a file with Google credentials if the given profile points to a GCS location
func GetPodWriter(cli kubernetes.Interface, ctx context.Context, namespace, podName, containerName string, profile *param.Profile) (*kube.PodWriter, error) {
	if profile.Location.Type == crv1alpha1.LocationTypeGCS {
		pw := kube.NewPodWriter(cli, consts.GoogleCloudCredsFilePath, bytes.NewBufferString(profile.Credential.KeyPair.Secret))
		if err := pw.Write(ctx, namespace, podName, containerName); err != nil {
			return nil, err
		}
		return pw, nil
	}
	return nil, nil
}

// CleanUpCredsFile is used to remove the file created by the given PodWriter
func CleanUpCredsFile(ctx context.Context, pw *kube.PodWriter, namespace, podName, containerName string) {
	if pw != nil {
		if err := pw.Remove(ctx, namespace, podName, containerName); err != nil {
			log.Error().WithContext(ctx).Print("Could not delete the temp file")
		}
	}
}

// FetchPodVolumes returns a map of PVCName->MountPath for a given pod
func FetchPodVolumes(pod string, tp param.TemplateParams) (map[string]string, error) {
	switch {
	case tp.Deployment != nil:
		if pvcToMountPath, ok := tp.Deployment.PersistentVolumeClaims[pod]; ok {
			return pvcToMountPath, nil
		}
		return nil, errors.New("Failed to find volumes for the Pod: " + pod)
	case tp.StatefulSet != nil:
		if pvcToMountPath, ok := tp.StatefulSet.PersistentVolumeClaims[pod]; ok {
			return pvcToMountPath, nil
		}
		return nil, errors.New("Failed to find volumes for the Pod: " + pod)
	default:
		return nil, errors.New("Invalid Template Params")
	}
}
