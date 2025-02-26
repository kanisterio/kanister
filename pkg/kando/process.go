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
	"io"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/grpclog"
)

const (
	processAddressFlagName = "address"
)

// grpclogLogger method taken from grpclog/loggerv2.go:newLoggerV2
// see also logging/logrus/grpclogger.go.
// grpclogger.go may be the best way to setup the loggers
// but kanister logging, while using logrus, does not seem to offer a straightforward
// interface for performing the interfacing in the grpclogger.go example.
func grpclogLogger(cmd *cobra.Command) grpclog.LoggerV2 {
	var infow, warnw, errorw io.Writer
	infow = io.Discard
	warnw = io.Discard
	errorw = io.Discard
	switch {
	case logrus.IsLevelEnabled(logrus.InfoLevel):
		infow = cmd.ErrOrStderr()
	case logrus.IsLevelEnabled(logrus.WarnLevel):
		warnw = cmd.ErrOrStderr()
	case logrus.IsLevelEnabled(logrus.ErrorLevel):
		errorw = cmd.ErrOrStderr()
	}
	return grpclog.NewLoggerV2(infow, warnw, errorw)
}

func newProcessCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process <command>",
		Short: "Manage kando processes",
	}

	cmd.AddCommand(newProcessServerCommand())
	cmd.AddCommand(newProcessClientCommand())
	cmd.PersistentFlags().StringP(processAddressFlagName, "a", "/tmp/kanister.sock", "The path of a unix socket of the process server")
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		err := setLogLevel(logLevel)
		if err != nil {
			return err
		}
		grpclog.SetLoggerV2(grpclogLogger(cmd))
		return nil
	}
	return cmd
}

func processAddressFlagValue(cmd *cobra.Command) (string, error) {
	a := cmd.Flag(processAddressFlagName).Value.String()
	_, err := net.ResolveUnixAddr("unix", a)
	return a, err
}
