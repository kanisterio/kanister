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

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	kankopia "github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	BackupDataUsingKopiaServerFuncName = "BackupDataUsingKopiaServer"
	// BackupDataUsingKopiaServerSnapshotTagsArg is the key used for returning snapshot tags
	BackupDataUsingKopiaServerSnapshotTagsArg = "snapshotTags"
)

type backupDataUsingKopiaServerFunc struct{}

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
	}
}

func (*backupDataUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]any) (map[string]any, error) {
	var (
		container   string
		err         error
		includePath string
		namespace   string
		pod         string
		tagsStr     string
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

	username := tp.RepositoryServer.Username
	hostname, userAccessPassphrase, err := hostNameAndUserPassPhraseFromRepoServer(userPassphrase)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch Hostname/User Passphrase from Secret")
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Kubernetes client")
	}

	snapInfo, err := backupDataUsingKopiaServer(
		cli,
		container,
		hostname,
		includePath,
		namespace,
		pod,
		tp.RepositoryServer.Address,
		fingerprint,
		username,
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

func backupDataUsingKopiaServer(
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
		UserPassword:    userPassphrase,
		ConfigFilePath:  configFile,
		LogDirectory:    logDirectory,
		CacheDirectory:  kopiacmd.DefaultCacheDirectory,
		Hostname:        hostname,
		ServerURL:       serverAddress,
		Fingerprint:     fingerprint,
		Username:        username,
		ContentCacheMB:  contentCacheMB,
		MetadataCacheMB: metadataCacheMB,
	})

	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
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
	stdout, stderr, err = kube.Exec(cli, namespace, pod, container, cmd, nil)
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

func hostNameAndUserPassPhraseFromRepoServer(userCreds string) (string, string, error) {
	var userAccessMap map[string]string
	if err := json.Unmarshal([]byte(userCreds), &userAccessMap); err != nil {
		return "", "", errors.Wrap(err, "Failed to unmarshal User Credentials Data")
	}

	var userPassPhrase string
	var hostName string
	for key, val := range userAccessMap {
		hostName = key
		userPassPhrase = val
	}

	decodedUserPassPhrase, err := base64.StdEncoding.DecodeString(userPassPhrase)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to Decode User Passphrase")
	}
	return hostName, string(decodedUserPassPhrase), nil
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
