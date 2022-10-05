package command

const (
	urlFlag = "--url"
)

type RepositoryServerCommandArgs struct {
	UserPassword    string
	ConfigFilePath  string
	LogDirectory    string
	CacheDirectory  string
	Hostname        string
	ServerURL       string
	Fingerprint     string
	Username        string
	ContentCacheMB  int
	MetadataCacheMB int
}

// RepositoryConnectServerCommand returns the kopia command for connecting to a remote repository on Kopia API server
func RepositoryConnectServerCommand(cmdArgs RepositoryServerCommandArgs) []string {
	args := commonArgs(&CommandArgs{
		RepoPassword:   cmdArgs.UserPassword,
		ConfigFilePath: cmdArgs.ConfigFilePath,
		LogDirectory:   cmdArgs.LogDirectory,
	}, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, serverSubCommand, noCheckForUpdatesFlag, noGrpcFlag)

	args = kopiaCacheArgs(args, cmdArgs.CacheDirectory, cmdArgs.ContentCacheMB, cmdArgs.MetadataCacheMB)

	if cmdArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}
	args = args.AppendLoggableKV(urlFlag, cmdArgs.ServerURL)

	args = args.AppendRedactedKV(serverCertFingerprint, cmdArgs.Fingerprint)

	return stringSliceCommand(args)
}
