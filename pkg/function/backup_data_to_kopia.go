package function

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/format"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	kopiarepo "github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	BackupDataToKopiaFuncName = "BackupDataToKopia"
	BackupDataTagsKeyArg      = "snapshotTags"
	// HostNameOption is the key for passing in hostname through Options map
	HostNameOption = "hostName"
	// UserNameOption is the key for passing in username through Options map
	UserNameOption = "userName"
	// KopiaFuncVersion is the version used by Kopia based Kanister functions for registration
	KopiaFuncVersion = "v1.0.0-alpha"
)

type backupDataToKopiaFunc struct{}

func init() {
	_ = kanister.Register(&backupDataToKopiaFunc{})
}

var _ kanister.Func = (*backupDataToKopiaFunc)(nil)

func (b backupDataToKopiaFunc) Name() string {
	return BackupDataToKopiaFuncName
}

func (b backupDataToKopiaFunc) RequiredArgs() []string {
	return []string{
		BackupDataNamespaceArg,
		BackupDataPodArg,
		BackupDataContainerArg,
		BackupDataIncludePathArg,
		BackupDataBackupArtifactPrefixArg,
		BackupDataEncryptionKeyArg,
	}
}

func (b backupDataToKopiaFunc) Arguments() []string {
	return []string{
		BackupDataNamespaceArg,
		BackupDataPodArg,
		BackupDataContainerArg,
		BackupDataIncludePathArg,
		BackupDataBackupArtifactPrefixArg,
		BackupDataEncryptionKeyArg,
		BackupDataTagsKeyArg,
	}
}

func (b backupDataToKopiaFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	//TODO implement me
	var namespace, pod, container, includePath, backupArtifactPrefix, encryptionKey, tagsStr string
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
	if err = Arg(args, BackupDataEncryptionKeyArg, &encryptionKey); err != nil {
		return nil, err
	}

	if err = OptArg(args, BackupDataTagsKeyArg, &tagsStr, ""); err != nil {
		return nil, err
	}

	var tags []string = nil
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, errors.Wrap(err, "Failed to validate Profile")
	}

	hostname, username, err := getHostAndUserNameFromOptions(tp.Options)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get hostname/username from Options")
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Kubernetes client")
	}

	ctx = field.Context(ctx, consts.PodNameKey, pod)
	ctx = field.Context(ctx, consts.ContainerNameKey, container)
	snapInfo, err := backupDataToKopia(
		ctx,
		cli,
		namespace,
		pod,
		container,
		backupArtifactPrefix,
		includePath,
		encryptionKey,
		hostname,
		username,
		tp,
		tags,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to backup data")
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

func backupDataToKopia(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	backupArtifactPrefix,
	includePath,
	encryptionKey,
	hostname,
	username string,
	tp param.TemplateParams,
	tags []string,
) (info *kopiacmd.SnapshotCreateInfo, err error) {
	pw, err := GetPodWriter(cli, ctx, namespace, pod, container, tp.Profile)
	if err != nil {
		return nil, err
	}
	defer CleanUpCredsFile(ctx, pw, namespace, pod, container)

	contentCacheMB, metadataCacheMB := kopiacmd.GetCacheSizeSettingsForSnapshot()

	profile, _ := json.Marshal(&param.KanisterProfile{Profile: tp.Profile})
	var location map[string][]byte
	err = json.Unmarshal(profile, &location)
	if err != nil {
		return nil, err
	}

	if err = kopiarepo.ConnectToOrCreateKopiaRepository(
		cli,
		namespace,
		pod,
		container,
		kopiacmd.RepositoryCommandArgs{
			CommandArgs: &kopiacmd.CommandArgs{
				RepoPassword:   encryptionKey,
				ConfigFilePath: kopiacmd.DefaultConfigFilePath,
				LogDirectory:   kopiacmd.DefaultLogDirectory,
			},
			CacheDirectory:  kopiacmd.DefaultCacheDirectory,
			Hostname:        hostname,
			Username:        username,
			ContentCacheMB:  contentCacheMB,
			MetadataCacheMB: metadataCacheMB,
			RepoPathPrefix:  backupArtifactPrefix,
			Location:        location,
		},
	); err != nil {
		return nil, err
	}

	cmd := kopiacmd.SnapshotCreate(
		kopiacmd.SnapshotCreateCommandArgs{
			PathToBackup: includePath,
			CommandArgs: &kopiacmd.CommandArgs{
				RepoPassword:   encryptionKey,
				ConfigFilePath: kopiacmd.DefaultConfigFilePath,
				LogDirectory:   kopiacmd.DefaultLogDirectory,
			},
			Tags:                   tags,
			ProgressUpdateInterval: 0,
		})
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
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

func getHostAndUserNameFromOptions(options map[string]string) (string, string, error) {
	var hostname, username string
	var ok bool
	if hostname, ok = options[HostNameOption]; !ok {
		return hostname, username, errors.New("Failed to find hostname option")
	}
	if username, ok = options[UserNameOption]; !ok {
		return hostname, username, errors.New("Failed to find username option")
	}
	return hostname, username, nil
}
