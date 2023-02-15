package function

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/kanisterio/kanister/pkg/log"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/format"
	kankopia "github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	BackupDataUsingKopiaServerFuncName = "BackupDataUsingKopiaServer"
	// HostNameOption is the key for passing in hostname through Options map
	HostNameOption = "hostName"
	// UserNameOption is the key for passing in username through Options map
	UserNameOption = "userName"
	// KopiaFuncVersion is the version used by Kopia based Kanister functions for registration
	KopiaFuncVersion = "v1.0.0-alpha"
	// BackupDataTagsKeyArg is the key used for returning snapshot tags
	BackupDataTagsKeyArg = "snapshotTags"
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
		//kankopia.KopiaAPIServerAddressArg,
		//kankopia.KopiaServerPassphraseArg,
		kankopia.KopiaUserPassphraseArg,
		kankopia.KopiaTLSCertSecretDataArg,
	}
}

func (*backupDataUsingKopiaServerFunc) Arguments() []string {
	return []string{
		BackupDataContainerArg,
		BackupDataIncludePathArg,
		BackupDataNamespaceArg,
		BackupDataPodArg,
		//kankopia.KopiaAPIServerAddressArg,
		//kankopia.KopiaServerPassphraseArg,
		kankopia.KopiaUserPassphraseArg,
		kankopia.KopiaTLSCertSecretDataArg,
		BackupDataTagsKeyArg,
	}
}

func (*backupDataUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]any) (map[string]any, error) {
	//TODO implement me
	var (
		container   string
		err         error
		includePath string
		namespace   string
		pod         string
		//serverAddress string
		//serverPassphrase string
		userPassphrase string
		cert           string
		tagsStr        string
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
	//if err = Arg(args, kankopia.KopiaAPIServerAddressArg, &serverAddress); err != nil {
	//	return nil, err
	//}
	//if err = Arg(args, kankopia.KopiaServerPassphraseArg, &serverPassphrase); err != nil {
	//	return nil, err
	//}
	if err = Arg(args, kankopia.KopiaUserPassphraseArg, &userPassphrase); err != nil {
		return nil, err
	}
	if err = Arg(args, kankopia.KopiaTLSCertSecretDataArg, &cert); err != nil {
		return nil, err
	}
	if err = OptArg(args, BackupDataTagsKeyArg, &tagsStr, ""); err != nil {
		return nil, err
	}

	var tags []string = nil
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}

	fingerprint, err := kankopia.ExtractFingerprintFromCertificateJSON(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch Kopia API Server Certificate Secret Data from blueprint")
	}

	username := tp.RepositoryServer.Username
	hostname, userAccessPassphrase, err := getHostNameAndUserPassPhraseFromRepoServer(userPassphrase)
	serverAddress := "https://" + tp.RepositoryServer.ServerInfo.ServiceName + ".kanister.svc.cluster.local:51515"

	log.Print("<--- User, Passphrase, Host and Server Address ---->", field.M{
		"Username":         username,
		"User Pass Phrase": userAccessPassphrase,
		"Hostname":         hostname,
		"Server Address":   serverAddress,
	})

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Kubernetes client")
	}

	ctx = field.Context(ctx, consts.PodNameKey, pod)
	ctx = field.Context(ctx, consts.ContainerNameKey, container)
	snapInfo, err := backupDataUsingKopiaServer(
		ctx,
		cli,
		container,
		hostname,
		includePath,
		namespace,
		pod,
		serverAddress,
		fingerprint,
		//serverPassphrase,
		username,
		userAccessPassphrase,
		tags,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to backup data to Kopia API server")
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
		FunctionOutputVersion:              KopiaFuncVersion,
	}
	return output, nil
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
	//serverPassphrase,
	username,
	userPassphrase string,
	tags []string,
) (info *kopiacmd.SnapshotCreateInfo, err error) {
	contentCacheMB, metadataCacheMB := kopiacmd.GetCacheSizeSettingsForSnapshot()
	configFile, logDirectory := kankopia.GetCustomConfigFileAndLogDirectory(hostname)

	// Check the status of Kopia API Server before connecting
	//cmd := kopiacmd.ServerStatus(
	//	kopiacmd.ServerStatusCommandArgs{
	//		CommandArgs: &kopiacmd.CommandArgs{
	//			RepoPassword:   "",
	//			ConfigFilePath: kopiacmd.DefaultConfigFilePath,
	//			LogDirectory:   kopiacmd.DefaultLogDirectory,
	//		},
	//		ServerAddress:  serverAddress,
	//		ServerUsername: kankopia.GetDefaultServerUsername(),
	//		ServerPassword: serverPassphrase,
	//		Fingerprint:    fingerprint,
	//	})
	//if err = kankopia.WaitTillCommandSucceed(ctx, cli, cmd, namespace, pod, container); err != nil {
	//	return nil, errors.Wrap(err, "Failed to establish connection to Kopia API server")
	//}

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
		return nil, errors.Wrap(err, "Failed to connect to Kopia API server")
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
		})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate snapshot create command")
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

func getHostNameAndUserPassPhraseFromRepoServer(userCreds string) (string, string, error) {
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

	decodedUserPassPhrase, _ := base64.StdEncoding.DecodeString(userPassPhrase)
	return hostName, string(decodedUserPassPhrase), nil

}
