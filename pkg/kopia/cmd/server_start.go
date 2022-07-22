// Copyright 2022 The Kanister Authors.
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

package cmd

// ServerStart returns the kopia command for starting the Kopia API Server
func ServerStart(
	configFilePath,
	logDirectory,
	serverAddress,
	tlsCertFile,
	tlsKeyFile,
	serverUsername,
	serverPassword string,
	autoGenerateCert,
	background bool,
) []string {
	args := commonArgs("", configFilePath, logDirectory, false)

	if autoGenerateCert {
		args = args.AppendLoggable(serverSubCommand, startSubCommand, tlsGenerateCertFlag)
	} else {
		args = args.AppendLoggable(serverSubCommand, startSubCommand)
	}
	args = args.AppendLoggableKV(addressFlag, serverAddress)
	args = args.AppendLoggableKV(tlsCertFilePath, tlsCertFile)
	args = args.AppendLoggableKV(tlsKeyFilePath, tlsKeyFile)
	args = args.AppendLoggableKV(serverUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverPassword)

	args = args.AppendLoggableKV(serverControlUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverControlPasswordFlag, serverPassword)

	// TODO: Remove when GRPC support is added
	args = args.AppendLoggable(noGrpcFlag)

	if background {
		// To start the server and run in the background
		args = args.AppendLoggable(redirectToDevNull, runInBackground)
	}

	return bashCommand(args)
}
