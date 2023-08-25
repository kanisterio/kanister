package validatingwebhook

import (
	"fmt"
	"os"
)

const WHCertsDir = "/var/run/webhook/serving-cert"

func IsCACertMounted() bool {
	if _, err := os.Stat(fmt.Sprintf("%s/%s", WHCertsDir, "tls.crt")); err != nil {
		return false
	}

	return true
}
