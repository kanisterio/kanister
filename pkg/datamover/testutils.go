// Copyright 2023 The Kanister Authors.
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

package datamover

import (
	"os/exec"
	"strings"

	"gopkg.in/check.v1"
)

func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func fingerprintFromTLSCert(c *check.C, tlsCert string) string {
	var args []string
	args = append(args, "openssl")
	args = append(args, "x509")
	args = append(args, "-fingerprint")
	args = append(args, "-noout")
	args = append(args, "-sha256")
	args = append(args, "-in")
	args = append(args, tlsCert)
	output := Command(c, args...)
	output = strings.TrimPrefix(output, "sha256 Fingerprint=")
	output = strings.ReplaceAll(output, ":", "")
	output = strings.ReplaceAll(output, "\n", "")
	return output
}

func readTLSCert(c *check.C, tlsCert string) string {
	var args []string
	args = append(args, "cat")
	args = append(args, tlsCert)
	output := Command(c, args...)
	return output
}

func Command(c *check.C, args ...string) string {
	c.Log(redactArgs(splitArgs(args)))
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	c.Log(string(out))
	c.Check(err, check.IsNil)

	return string(out)
}

func redactArgs(args []string) []string {
	const redacted = "<redacted>"
	var redactNext bool
	r := make([]string, 0, len(args))
	for _, a := range args {
		if redactNext {
			r = append(r, redacted)
			redactNext = false
			continue
		}
		switch a {
		case "--access-key", "--secret-access-key", "--password", "--storage-account", "--storage-key":
			redactNext = true
		}
		if strings.HasPrefix(a, "--access-key=") ||
			strings.HasPrefix(a, "--secret-access-key=") ||
			strings.HasPrefix(a, "--password=") ||
			strings.HasPrefix(a, "--storage-account=") ||
			strings.HasPrefix(a, "--storage-key=") {
			p := strings.Split(a, "=")
			a = p[0] + "=" + redacted
		}
		r = append(r, a)
	}
	return r
}

func splitArgs(args []string) []string {
	var r []string
	for _, a := range args {
		r = append(r, strings.Fields(a)...)
	}
	return r
}
