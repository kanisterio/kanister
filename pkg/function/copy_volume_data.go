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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kanisterio/errkit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/restic"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// CopyVolumeDataFuncName gives the function name
	CopyVolumeDataFuncName                     = "CopyVolumeData"
	CopyVolumeDataMountPoint                   = "/mnt/vol_data/%s"
	CopyVolumeDataJobPrefix                    = "copy-vol-data-"
	CopyVolumeDataNamespaceArg                 = "namespace"
	CopyVolumeDataVolumeArg                    = "volume"
	CopyVolumeDataArtifactPrefixArg            = "dataArtifactPrefix"
	CopyVolumeDataOutputBackupID               = "backupID"
	CopyVolumeDataOutputBackupRoot             = "backupRoot"
	CopyVolumeDataOutputBackupArtifactLocation = "backupArtifactLocation"
	CopyVolumeDataEncryptionKeyArg             = "encryptionKey"
	CopyVolumeDataOutputBackupTag              = "backupTag"
	CopyVolumeDataPodOverrideArg               = "podOverride"
	CopyVolumeDataOutputBackupFileCount        = "fileCount"
	CopyVolumeDataOutputBackupSize             = "size"
	CopyVolumeDataOutputPhysicalSize           = "phySize"
)

func init() {
	_ = kanister.Register(&copyVolumeDataFunc{})
}

var _ kanister.Func = (*copyVolumeDataFunc)(nil)

type copyVolumeDataFunc struct {
	progressPercent string
}

func (*copyVolumeDataFunc) Name() string {
	return CopyVolumeDataFuncName
}

func copyVolumeData(
	ctx context.Context,
	cli kubernetes.Interface,
	tp param.TemplateParams,
	namespace,
	pvcName,
	targetPath,
	encryptionKey string,
	insecureTLS bool,
	podOverride map[string]interface{},
	annotations,
	labels map[string]string,
) (map[string]interface{}, error) {
	// Validate PVC exists
	pvc, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to retrieve PVC.", "namespace", namespace, "name", pvcName)
	}

	// Create a pod with PVCs attached
	mountPoint := fmt.Sprintf(CopyVolumeDataMountPoint, pvcName)
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: CopyVolumeDataJobPrefix,
		Image:        consts.GetKanisterToolsImage(),
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes: map[string]kube.VolumeMountOptions{pvcName: {
			MountPath: mountPoint,
			ReadOnly:  kube.PVCContainsReadOnlyAccessMode(pvc),
		}},
		PodOverride: podOverride,
		Annotations: annotations,
		Labels:      labels,
	}

	// Apply the registered ephemeral pod changes.
	ephemeral.PodOptions.Apply(options)

	pr := kube.NewPodRunner(cli, options)
	podFunc := copyVolumeDataPodFunc(cli, tp, mountPoint, targetPath, encryptionKey, insecureTLS)
	return pr.Run(ctx, podFunc)
}

func copyVolumeDataPodFunc(
	cli kubernetes.Interface,
	tp param.TemplateParams,
	mountPoint,
	targetPath,
	encryptionKey string,
	insecureTLS bool,
) func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
		// Wait for pod to reach running state
		if err := pc.WaitForPodReady(ctx); err != nil {
			return nil, errkit.Wrap(err, "Failed while waiting for Pod to be ready", "pod", pc.PodName())
		}

		remover, err := MaybeWriteProfileCredentials(ctx, pc, tp.Profile)
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to write credentials to Pod", "pod", pc.PodName())
		}

		// Parent context could already be dead, so removing file within new context
		defer remover.Remove(context.Background()) //nolint:errcheck

		pod := pc.Pod()
		// Get restic repository
		if err := restic.GetOrCreateRepository(
			ctx,
			cli,
			pod.Namespace,
			pod.Name,
			pod.Spec.Containers[0].Name,
			targetPath,
			encryptionKey,
			insecureTLS,
			tp.Profile,
		); err != nil {
			return nil, err
		}
		// Copy data to object store
		backupTag := rand.String(10)

		// Build backup command that changes to mount point directory first
		// to avoid absolute path issues during restore
		cmd, err := buildBackupCommandWithCD(tp.Profile, targetPath, backupTag, mountPoint, encryptionKey, insecureTLS)
		if err != nil {
			return nil, err
		}

		ex, err := pc.GetCommandExecutor()
		if err != nil {
			return nil, err
		}
		stdout, _, err := ExecAndLog(ctx, ex, cmd, pod)
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to create and upload backup")
		}
		// Get the snapshot ID from log
		backupID := restic.SnapshotIDFromBackupLog(stdout)
		if backupID == "" {
			return nil, errkit.New(fmt.Sprintf("Failed to parse the backup ID from logs, backup logs %s", stdout))
		}
		fileCount, backupSize, phySize := restic.SnapshotStatsFromBackupLog(stdout)
		if backupSize == "" {
			log.Debug().Print("Could not parse backup stats from backup log")
		}
		return map[string]interface{}{
				CopyVolumeDataOutputBackupID:               backupID,
				CopyVolumeDataOutputBackupRoot:             mountPoint,
				CopyVolumeDataOutputBackupArtifactLocation: targetPath,
				CopyVolumeDataOutputBackupTag:              backupTag,
				CopyVolumeDataOutputBackupFileCount:        fileCount,
				CopyVolumeDataOutputBackupSize:             backupSize,
				CopyVolumeDataOutputPhysicalSize:           phySize,
				FunctionOutputVersion:                      kanister.DefaultVersion,
			},
			nil
	}
}

func (c *copyVolumeDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	c.progressPercent = progress.StartedPercent
	defer func() { c.progressPercent = progress.CompletedPercent }()

	var namespace, vol, targetPath, encryptionKey string
	var err error
	var bpAnnotations, bpLabels map[string]string
	var insecureTLS bool
	if err = Arg(args, CopyVolumeDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataVolumeArg, &vol); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataArtifactPrefixArg, &targetPath); err != nil {
		return nil, err
	}
	if err = OptArg(args, CopyVolumeDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	if err = OptArg(args, InsecureTLS, &insecureTLS, false); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
		return nil, err
	}
	podOverride, err := GetPodSpecOverride(tp, args, CopyVolumeDataPodOverrideArg)
	if err != nil {
		return nil, err
	}

	annotations := bpAnnotations
	labels := bpLabels
	if tp.PodAnnotations != nil {
		// merge the actionset annotations with blueprint annotations
		var actionSetAnn ActionSetAnnotations = tp.PodAnnotations
		annotations = actionSetAnn.MergeBPAnnotations(bpAnnotations)
	}

	if tp.PodLabels != nil {
		// merge the actionset labels with blueprint labels
		var actionSetLabels ActionSetLabels = tp.PodLabels
		labels = actionSetLabels.MergeBPLabels(bpLabels)
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, errkit.Wrap(err, "Failed to validate Profile")
	}

	targetPath = ResolveArtifactPrefix(targetPath, tp.Profile)

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create Kubernetes client")
	}
	return copyVolumeData(
		ctx,
		cli,
		tp,
		namespace,
		vol,
		targetPath,
		encryptionKey,
		insecureTLS,
		podOverride,
		annotations,
		labels,
	)
}

// buildBackupCommandWithCD creates a backup command that changes to the mount directory first
// to ensure relative paths in the backup, avoiding absolute path issues during restore
func buildBackupCommandWithCD(profile *param.Profile, repository, backupTag, mountPoint, encryptionKey string, insecureTLS bool) ([]string, error) {
	// Get the base restic command args - we need to duplicate this logic since resticArgs is not exported
	var cmd []string
	var err error
	switch profile.Location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
		cmd, err = buildS3Args(profile, repository)
	case crv1alpha1.LocationTypeGCS:
		cmd = buildGCSArgs(profile, repository)
	case crv1alpha1.LocationTypeAzure:
		cmd, err = buildAzureArgs(profile, repository)
	default:
		return nil, errkit.New(fmt.Sprintf("Unsupported type '%s' for the location", profile.Location.Type))
	}
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to get arguments")
	}

	// Add password and restic command
	cmd = append(cmd, fmt.Sprintf("export %s=%s", restic.ResticPassword, encryptionKey))

	// Build backup command parts
	backupArgs := []string{restic.ResticCommand, "backup", "--tag", backupTag, "."}
	if insecureTLS {
		backupArgs = append(backupArgs, "--insecure-tls")
	}

	// Combine everything: environment setup, cd to mount point, then run backup
	cmd = append(cmd, fmt.Sprintf("cd %s", mountPoint), strings.Join(backupArgs, " "))
	command := strings.Join(cmd, "\n")

	// Return wrapped command
	return []string{"bash", "-o", "errexit", "-o", "pipefail", "-c", command}, nil
}

// Helper functions to build cloud provider args (simplified versions of restic internal functions)
func buildS3Args(profile *param.Profile, repository string) ([]string, error) {
	s3Endpoint := "s3.amazonaws.com"
	if profile.Location.Endpoint != "" {
		s3Endpoint = profile.Location.Endpoint
	}
	if strings.HasSuffix(s3Endpoint, "/") {
		s3Endpoint = strings.TrimRight(s3Endpoint, "/")
	}

	var args []string
	switch profile.Credential.Type {
	case param.CredentialTypeKeyPair:
		args = []string{
			fmt.Sprintf("export %s=%s", location.AWSAccessKeyID, profile.Credential.KeyPair.ID),
			fmt.Sprintf("export %s=%s", location.AWSSecretAccessKey, profile.Credential.KeyPair.Secret),
		}
	case param.CredentialTypeSecret:
		creds, err := secrets.ExtractAWSCredentials(context.Background(), profile.Credential.Secret, aws.AssumeRoleDurationDefault)
		if err != nil {
			return nil, err
		}
		args = []string{
			fmt.Sprintf("export %s=%s", location.AWSAccessKeyID, creds.AccessKeyID),
			fmt.Sprintf("export %s=%s", location.AWSSecretAccessKey, creds.SecretAccessKey),
		}
		if creds.SessionToken != "" {
			args = append(args, fmt.Sprintf("export %s=%s", location.AWSSessionToken, creds.SessionToken))
		}
	default:
		return nil, errkit.New(fmt.Sprintf("Unsupported type '%s' for credentials", profile.Credential.Type))
	}
	args = append(args, fmt.Sprintf("export %s=s3:%s/%s", restic.ResticRepository, s3Endpoint, repository))
	return args, nil
}

func buildGCSArgs(profile *param.Profile, repository string) []string {
	return []string{
		fmt.Sprintf("export %s=%s", location.GoogleProjectID, profile.Credential.KeyPair.ID),
		fmt.Sprintf("export %s=%s", location.GoogleCloudCreds, consts.GoogleCloudCredsFilePath),
		fmt.Sprintf("export %s=gs:%s/", restic.ResticRepository, strings.Replace(repository, "/", ":/", 1)),
	}
}

func buildAzureArgs(profile *param.Profile, repository string) ([]string, error) {
	var storageAccountID, storageAccountKey string
	switch profile.Credential.Type {
	case param.CredentialTypeKeyPair:
		storageAccountID = profile.Credential.KeyPair.ID
		storageAccountKey = profile.Credential.KeyPair.Secret
	case param.CredentialTypeSecret:
		creds, err := secrets.ExtractAzureCredentials(profile.Credential.Secret)
		if err != nil {
			return nil, err
		}
		storageAccountID = creds.StorageAccount
		storageAccountKey = creds.StorageKey
	}

	return []string{
		fmt.Sprintf("export %s=%s", location.AzureStorageAccount, storageAccountID),
		fmt.Sprintf("export %s=%s", location.AzureStorageKey, storageAccountKey),
		fmt.Sprintf("export %s=azure:%s/", restic.ResticRepository, strings.Replace(repository, "/", ":/", 1)),
	}, nil
}

func (*copyVolumeDataFunc) RequiredArgs() []string {
	return []string{
		CopyVolumeDataNamespaceArg,
		CopyVolumeDataVolumeArg,
		CopyVolumeDataArtifactPrefixArg,
	}
}

func (*copyVolumeDataFunc) Arguments() []string {
	return []string{
		CopyVolumeDataNamespaceArg,
		CopyVolumeDataVolumeArg,
		CopyVolumeDataArtifactPrefixArg,
		CopyVolumeDataEncryptionKeyArg,
		InsecureTLS,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (c *copyVolumeDataFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(c.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(c.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(c.RequiredArgs(), args)
}

func (c *copyVolumeDataFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    c.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
