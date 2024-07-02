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

package kanctl

import (
	"os"

	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/version"
)

const (
	namespaceFlagName = "namespace"
	verboseFlagName   = "verbose"
)

var (
	// Verbose indicates whether verbose output should be displayed
	Verbose bool
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		if Verbose {
			log.WithError(err).Print("Kanctl failed to execute")
		}

		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	// RootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     "kanctl [common options...] <command>",
		Short:   "A set of helpers to help with management of Kanister custom resources",
		Version: version.VersionString(),
	}
	rootCmd.PersistentFlags().StringP(namespaceFlagName, "n", "", "Override namespace obtained from kubectl context")
	rootCmd.PersistentFlags().BoolVar(&Verbose, verboseFlagName, false, "Display verbose output")
	rootCmd.AddCommand(newValidateCommand())
	rootCmd.AddCommand(newCreateCommand())
	return rootCmd
}

func resolveNamespace(cmd *cobra.Command) (string, error) {
	if ns := cmd.Flag(namespaceFlagName).Value.String(); ns != "" {
		return ns, nil
	}
	return kube.ConfigNamespace()
}

func initializeClients() (kubernetes.Interface, versioned.Interface, osversioned.Interface, error) {
	config, err := kube.LoadConfig()
	if err != nil {
		return nil, nil, nil, err
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "could not get the kubernetes client")
	}

	osCli, err := osversioned.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "could not get openshift client")
	}

	crCli, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "could not get the CRD client")
	}
	return cli, crCli, osCli, nil
}
