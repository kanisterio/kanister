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
	"fmt"
	"os"

	"github.com/kanisterio/errkit"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/grpclog"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/version"
)

var logLevel string

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

	rootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", logrus.WarnLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		err := setLogLevel(logLevel)
		if err != nil {
			return err
		}
		grpclog.SetLoggerV2(grpclogLogger(cmd))
		return nil
	}

	rootCmd.AddCommand(newLocationCommand())
	rootCmd.AddCommand(newOutputCommand())
	rootCmd.AddCommand(newChronicleCommand())
	rootCmd.AddCommand(newStreamCommand())
	rootCmd.AddCommand(newProcessCommand())
	return rootCmd
}

func setLogLevel(v string) error {
	lgl, err := logrus.ParseLevel(v)
	if err != nil {
		return errkit.Wrap(err, fmt.Sprintf("Invalid log level: %s", v))
	}
	// set application logger log level. (kanister/log/log.go)
	log.SetLevel(log.Level(lgl))
	// set "std" logrus logger.  GRPC uses this (logrus/exported)
	logrus.SetLevel(lgl)
	return nil
}
