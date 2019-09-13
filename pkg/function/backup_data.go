// Copyright 2019 The Kanister Authors.
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
	"bytes"
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// BackupDataNamespaceArg provides the namespace
	BackupDataNamespaceArg = "namespace"
	// BackupDataPodArg provides the pod connected to the data volume
	BackupDataPodArg = "pod"
	// BackupDataContainerArg provides the container on which the backup is taken
	BackupDataContainerArg = "container"
	// BackupDataIncludePathArg provides the path of the volume or sub-path for required backup
	BackupDataIncludePathArg = "includePath"
	// BackupDataBackupArtifactPrefixArg provides the path to store artifacts on the object store
	BackupDataBackupArtifactPrefixArg = "backupArtifactPrefix"
	// BackupDataEncryptionKeyArg provides the encryption key to be used for backups
	BackupDataEncryptionKeyArg = "encryptionKey"
	// BackupDataOutputBackupID is the key used for returning backup ID output
	BackupDataOutputBackupID = "backupID"
	// BackupDataOutputBackupTag is the key used for returning backupTag output
	BackupDataOutputBackupTag = "backupTag"
)

func init() {
	kanister.Register(&backupDataFunc{})
}

var _ kanister.Func = (*backupDataFunc)(nil)

type backupDataFunc struct{}

func (*backupDataFunc) Name() string {
	return "BackupData"
}

func validateProfile(profile *param.Profile) error {
	if profile == nil {
		return errors.New("Profile must be non-nil")
	}
	if profile.Credential.Type != param.CredentialTypeKeyPair {
		return errors.New("Credential type not supported")
	}
	if len(profile.Credential.KeyPair.ID) == 0 {
		return errors.New("Access key ID is not set")
	}
	if len(profile.Credential.KeyPair.Secret) == 0 {
		return errors.New("Secret access key is not set")
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

func (*backupDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, pod, container, includePath, backupArtifactPrefix, encryptionKey string
	var err error
	if err = Arg(args, BackupDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataPodArg, &pod); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataContainerArg, &container); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataIncludePathArg, &includePath); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataBackupArtifactPrefixArg, &backupArtifactPrefix); err != nil {
		return nil, err
	}
	if err = OptArg(args, BackupDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	ctx = field.Context(ctx, field.PodNameKey, pod)
	ctx = field.Context(ctx, field.ContainerNameKey, container)
	// Validate the Profile
	if err = validateProfile(tp.Profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to validate Profile")
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	backupID, backupTag, err := backupData(ctx, cli, namespace, pod, container, backupArtifactPrefix, includePath, encryptionKey, tp)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to backup data")
	}
	output := map[string]interface{}{
		BackupDataOutputBackupID:  backupID,
		BackupDataOutputBackupTag: backupTag,
	}
	return output, nil
}

func (*backupDataFunc) RequiredArgs() []string {
	return []string{BackupDataNamespaceArg, BackupDataPodArg, BackupDataContainerArg,
		BackupDataIncludePathArg, BackupDataBackupArtifactPrefixArg}
}

func backupData(ctx context.Context, cli kubernetes.Interface, namespace, pod, container, backupArtifactPrefix, includePath, encryptionKey string, tp param.TemplateParams) (string, string, error) {
	pw, err := getPodWriter(cli, ctx, namespace, pod, container, tp.Profile)
	if err != nil {
		return "", "", err
	}
	defer cleanUpCredsFile(ctx, pw, namespace, pod, container)
	if err = restic.GetOrCreateRepository(cli, namespace, pod, container, backupArtifactPrefix, encryptionKey, tp.Profile); err != nil {
		return "", "", err
	}

	// Create backup and dump it on the object store
	backupTag := rand.String(10)
	cmd := restic.BackupCommandByTag(tp.Profile, backupArtifactPrefix, backupTag, includePath, encryptionKey)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to create and upload backup")
	}
	// Get the snapshot ID from log
	backupID := restic.SnapshotIDFromBackupLog(stdout)
	if backupID == "" {
		return "", "", errors.New("Failed to parse the backup ID from logs")
	}
	return backupID, backupTag, nil
}

func getPodWriter(cli kubernetes.Interface, ctx context.Context, namespace, podName, containerName string, profile *param.Profile) (*kube.PodWriter, error) {
	if profile.Location.Type == crv1alpha1.LocationTypeGCS {
		pw := kube.NewPodWriter(cli, restic.GoogleCloudCredsFilePath, bytes.NewBufferString(profile.Credential.KeyPair.Secret))
		if err := pw.Write(ctx, namespace, podName, containerName); err != nil {
			return nil, err
		}
		return pw, nil
	}
	return nil, nil
}
func cleanUpCredsFile(ctx context.Context, pw *kube.PodWriter, namespace, podName, containerName string) {
	if pw != nil {
		if err := pw.Remove(ctx, namespace, podName, containerName); err != nil {
			log.WithContext(ctx).Error("Could not delete the temp file")
		}
	}
}
