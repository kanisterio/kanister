package kanctl

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
)

const (
	namespaceFlagName = "namespace"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		log.Errorf("%+v", err)
	}
}

func newRootCommand() *cobra.Command {
	// RootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     "kanctl [common options...] <command>",
		Short:   "A set of helpers to help with creating ActionSets",
		Version: kanister.VERSION,
	}
	rootCmd.PersistentFlags().StringP(namespaceFlagName, "n", "", "Override namespace obtained from kubectl context")
	rootCmd.AddCommand(newPerformFromCommand())
	rootCmd.AddCommand(newValidateCommand())
	return rootCmd
}

func resolveNamespace(cmd *cobra.Command) (string, error) {
	if ns := cmd.Flag(namespaceFlagName).Value.String(); ns != "" {
		return ns, nil
	}
	return kube.ConfigNamespace()
}

func initializeClients() (kubernetes.Interface, versioned.Interface, error) {
	config, err := kube.LoadConfig()
	if err != nil {
		return nil, nil, err
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get the kubernetes client")
	}
	crCli, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get the CRD client")
	}
	return cli, crCli, nil
}
