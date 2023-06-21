// Copyright 2020 The Kanister Authors.
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

package kando

import (
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/version"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		log.WithError(err).Print("Kando failed to execute")
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	// RootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     "kando <command>",
		Short:   "A set of tools used from Kanister Blueprints",
		Version: version.VersionString(),
	}

	var v string
	rootCmd.PersistentFlags().StringVarP(&v, "verbosity", "v", logrus.WarnLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		return setLogLevel(v)
	}

	rootCmd.AddCommand(newLocationCommand())
	rootCmd.AddCommand(newOutputCommand())
	rootCmd.AddCommand(newChronicleCommand())
	rootCmd.AddCommand(newStreamCommand())
	return rootCmd
}

func setLogLevel(v string) error {
	l, err := logrus.ParseLevel(v)
	if err != nil {
		return errors.Wrap(err, "Invalid log level: "+v)
	}
	logrus.SetLevel(l)
	return nil
}
