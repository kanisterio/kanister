package test

import "strings"

var redactedFlags = []string{
	"--password",
	"--user-password",
	"--server-password",
	"--server-control-password",
	"--server-cert-fingerprint",
}

// RedactCLI redacts sensitive information from the CLI command for tests.
func RedactCLI(cli []string) string {
	redactedCLI := make([]string, len(cli))
	for i, arg := range cli {
		redactField := ""
		for _, rf := range redactedFlags {
			if strings.HasPrefix(arg, rf) {
				redactField = rf
				break
			}
		}
		if len(redactField) > 0 {
			redactedCLI[i] = redactField + "=<****>"
		} else {
			redactedCLI[i] = arg
		}
	}
	return strings.Join(redactedCLI, " ")
}
