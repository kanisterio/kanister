package kanctl

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	kanister "github.com/kanisterio/kanister/pkg"
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
	return rootCmd
}

func resolveNamespace(cmd *cobra.Command) (string, error) {
	if ns := cmd.Flag(namespaceFlagName).Value.String(); ns != "" {
		return ns, nil
	}
	return kube.ConfigNamespace()
}
