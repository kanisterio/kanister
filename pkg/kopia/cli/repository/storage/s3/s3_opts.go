// Copyright 2024 The Kanister Authors.
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

package s3

import (
	"github.com/kanisterio/safecli/command"
)

var (
	subcmdS3 = command.NewArgument("s3")
)

// optBucket creates a new bucket option with a given name.
func optBucket(name string) command.Applier {
	return command.NewOptionWithArgument("--bucket", name)
}

// optEndpoint creates a new endpoint option with a given endpoint.
func optEndpoint(endpoint string) command.Applier {
	return command.NewOptionWithArgument("--endpoint", endpoint)
}

// optPrefix creates a new prefix option with a given prefix.
func optPrefix(prefix string) command.Applier {
	return command.NewOptionWithArgument("--prefix", prefix)
}

// optRegion creates a new region option with a given region.
func optRegion(region string) command.Applier {
	return command.NewOptionWithArgument("--region", region)
}

// optDisableTLS creates a new disable TLS option with a given value.
func optDisableTLS(disable bool) command.Applier {
	return command.NewOption("--disable-tls", disable)
}

// optDisableTLSVerify creates a new disable TLS verification option with a given value.
func optDisableTLSVerify(disable bool) command.Applier {
	return command.NewOption("--disable-tls-verification", disable)
}