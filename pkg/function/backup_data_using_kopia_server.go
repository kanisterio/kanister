// Copyright 2023 The Kanister Authors.
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
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/format"
	kankopia "github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	BackupDataUsingKopiaServerFuncName = "BackupDataUsingKopiaServer"
	// BackupDataUsingKopiaServerSnapshotTagsArg is the key used for returning snapshot tags
	BackupDataUsingKopiaServerSnapshotTagsArg = "snapshotTags"
	// KopiaRepositoryServerUserHostname is the key used for returning the hostname of the user
	KopiaRepositoryServerUserHostname = "repositoryServerUserHostname"
)

type backupDataUsingKopiaServerFunc struct {
	progressPercent string
}

func init() {
	err := kanister.Register(&backupDataUsingKopiaServerFunc{})
	if err != nil {
		return
	}
}

var _ kanister.Func = (*backupDataUsingKopiaServerFunc)(nil)

func (*backupDataUsingKopiaServerFunc) Name() string {
	return BackupDataUsingKopiaServerFuncName
}

func (*backupDataUsingKopiaServerFunc) RequiredArgs() []string {
	return []string{
		BackupDataContainerArg,
		BackupDataIncludePathArg,
		BackupDataNamespaceArg,
		BackupDataPodArg,
	}
}

func (*backupDataUsingKopiaServerFunc) Arguments() []string {
	return []string{
		BackupDataContainerArg,
		BackupDataIncludePathArg,
		BackupDataNamespaceArg,
		BackupDataPodArg,
		BackupDataUsingKopiaServerSnapshotTagsArg,
		KopiaRepositoryServerUserHostname,
	}
}

func (b *backupDataUsingKopiaServerFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(b.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(b.RequiredArgs(), args)
}

func (b *backupDataUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]any) (map[string]any, error) {
	// Set progress percent
	b.progressPercent = progress.StartedPercent
	defer func() { b.progressPercent = progress.CompletedPercent }()

	var (
		container    string
		err          error
		includePath  string
		namespace    string
		pod          string
		tagsStr      string
		userHostname string
	)
	if err = Arg(args, BackupDataContainerArg, &container); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataIncludePathArg, &includePath); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataPodArg, &pod); err != nil {
		return nil, err
	}
	if err = OptArg(args, BackupDataUsingKopiaServerSnapshotTagsArg, &tagsStr, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, KopiaRepositoryServerUserHostname, &userHostname, ""); err != nil {
		return nil, err
	}

	var tags []string = nil
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}

	userPassphrase, cert, err := userCredentialsAndServerTLS(&tp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch User Credentials/Certificate Data from Template Params")
	}

	fingerprint, err := kankopia.ExtractFingerprintFromCertificateJSON(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch Kopia API Server Certificate Secret Data from Certificate")
	}

	hostname, userAccessPassphrase, err := hostNameAndUserPassPhraseFromRepoServer(userPassphrase, userHostname)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch Hostname/User Passphrase from Secret")
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Kubernetes client")
	}

	snapInfo, err := backupDataUsingKopiaServer(
		ctx,
		cli,
		container,
		hostname,
		includePath,
		namespace,
		pod,
		tp.RepositoryServer.Address,
		fingerprint,
		tp.RepositoryServer.Username,
		userAccessPassphrase,
		tags,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to backup data using Kopia Repository Server")
	}

	var logSize, phySize int64
	if snapInfo.Stats != nil {
		stats := snapInfo.Stats
		logSize = stats.SizeHashedB + stats.SizeCachedB
		phySize = stats.SizeUploadedB
	}

	output := map[string]any{
		BackupDataOutputBackupID:           snapInfo.SnapshotID,
		BackupDataOutputBackupSize:         humanize.Bytes(uint64(logSize)),
		BackupDataOutputBackupPhysicalSize: humanize.Bytes(uint64(phySize)),
	}
	return output, nil
}

func (b *backupDataUsingKopiaServerFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    b.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

func backupDataUsingKopiaServer(
	ctx context.Context,
	cli kubernetes.Interface,
	container,
	hostname,
	includePath,
	namespace,
	pod,
	serverAddress,
	fingerprint,
	username,
	userPassphrase string,
	tags []string,
) (info *kopiacmd.SnapshotCreateInfo, err error) {
	contentCacheMB, metadataCacheMB := kopiacmd.GetCacheSizeSettingsForSnapshot()
	configFile, logDirectory := kankopia.CustomConfigFileAndLogDirectory(hostname)

	cmd := kopiacmd.RepositoryConnectServerCommand(kopiacmd.RepositoryServerCommandArgs{
		UserPassword:   userPassphrase,
		ConfigFilePath: configFile,
		LogDirectory:   logDirectory,
		CacheDirectory: kopiacmd.DefaultCacheDirectory,
		Hostname:       hostname,
		ServerURL:      serverAddress,
		Fingerprint:    fingerprint,
		Username:       username,
		CacheArgs: kopiacmd.CacheArgs{
			ContentCacheLimitMB:  contentCacheMB,
			MetadataCacheLimitMB: metadataCacheMB,
		},
	})

	stdout, stderr, err := kube.Exec(ctx, cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to Kopia Repository Server")
	}

	cmd = kopiacmd.SnapshotCreate(
		kopiacmd.SnapshotCreateCommandArgs{
			PathToBackup: includePath,
			CommandArgs: &kopiacmd.CommandArgs{
				RepoPassword:   "",
				ConfigFilePath: configFile,
				LogDirectory:   logDirectory,
			},
			Tags:                   tags,
			ProgressUpdateInterval: 0,
			Parallelism:            utils.GetEnvAsIntOrDefault(kankopia.DataStoreParallelUploadName, kankopia.DefaultDataStoreParallelUpload),
		})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to construct snapshot create command")
	}
	stdout, stderr, err = kube.Exec(ctx, cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)

	message := "Failed to create and upload backup"
	if err != nil {
		if strings.Contains(err.Error(), kerrors.ErrCodeOutOfMemoryStr) {
			message = message + ": " + kerrors.ErrOutOfMemoryStr
		}
		return nil, errors.Wrap(err, message)
	}
	// Parse logs and return snapshot IDs and stats
	return kopiacmd.ParseSnapshotCreateOutput(stdout, stderr)
}

func hostNameAndUserPassPhraseFromRepoServer(userCreds, hostname string) (string, string, error) {
	var userAccessMap map[string]string
	if err := json.Unmarshal([]byte(userCreds), &userAccessMap); err != nil {
		return "", "", errors.Wrap(err, "Failed to unmarshal User Credentials Data")
	}

	// Check if hostname provided exists in the User Access Map
	if hostname != "" {
		err := checkHostnameExistsInUserAccessMap(userAccessMap, hostname)
		if err != nil {
			return "", "", errors.Wrap(err, "Failed to find hostname in the User Access Map")
		}
	}

	// Set First Value of hostname and passphrase from the User Access Map
	// Or if hostname provided by the user, set the hostname and password for hostname provided
	var userPassphrase string
	for key, val := range userAccessMap {
		if hostname == "" || hostname == key {
			hostname = key
			userPassphrase = val
			break
		}
	}

	decodedUserPassphrase, err := base64.StdEncoding.DecodeString(userPassphrase)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to Decode User Passphrase")
	}
	return hostname, string(decodedUserPassphrase), nil
}

func userCredentialsAndServerTLS(tp *param.TemplateParams) (string, string, error) {
	userCredJSON, err := json.Marshal(tp.RepositoryServer.Credentials.ServerUserAccess.Data)
	if err != nil {
		return "", "", errors.Wrap(err, "Error marshalling User Credentials Data")
	}
	certJSON, err := json.Marshal(tp.RepositoryServer.Credentials.ServerTLS.Data)
	if err != nil {
		return "", "", errors.Wrap(err, "Error marshalling Certificate Data")
	}
	return string(userCredJSON), string(certJSON), nil
}

func checkHostnameExistsInUserAccessMap(userAccessMap map[string]string, hostname string) error {
	// check if hostname that is provided by the user exists in the user access map
	if _, ok := userAccessMap[hostname]; !ok {
		return errors.New("hostname provided in the repository server CR does not exist in the user access map")
	}
	return nil
}
