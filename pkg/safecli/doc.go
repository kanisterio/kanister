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

// Package safecli provides functionality for building, redacting, and logging command-line arguments.
// It is designed to handle sensitive values securely
// while also providing utility for CLI construction and logging.
package safecli

// The package offers two concrete builder functions:
// - NewBuilder: Creates a new CLI builder, allowing for flexible CLI creation.
// - NewLogger: Creates a new CLI logger for safely CLI logging.

// Usage example:
//
// package main
//
// import (
//     "fmt"
//     "safecli"
// )
//
// func main() {
//     // Create a new command builder
//     zipcli := safecli.NewBuilder("zip").
//         AppendLoggableKV(
//             "--temp-path", "/tmp",
//             "--exclude", "*.log",
//         ).
//         AppendRedactedKV("-p", "password123").
//         AppendLoggable(
//             "project_backup.zip",
//             "~/home/user/project")
//
//     fmt.Println("Builder:", zipcli)
//     // Output: Builder: [zip --temp-path=/tmp --exclude=*.log -p=<****> project_backup.zip ~/home/user/project]
//     // The fmt.Println call implicitly invokes the String() method on zipcli,
//     // which returns a log string representation of the command.
//     // This is similar to calling logger.Log() as shown below.
//
//     // Build the command.
//     command := zipcli.Build()
//     fmt.Println("Command:", command)
//     // Output: Command: [zip --temp-path=/tmp --exclude=*.log -p=password123 project_backup.zip ~/home/user/project]
//
//     // Log the command with sensitive data redacted.
//     logger := safecli.NewLogger(zipcli)
//     logOutput := logger.Log()
//
//     fmt.Println("Log:", logOutput)
//     // Output: Log: zip --temp-path=/tmp --exclude=*.log -p=<****> project_backup.zip ~/home/user/project
// }
//
