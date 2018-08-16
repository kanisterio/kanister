package restic

import (
	"crypto/sha256"
	"fmt"
)

const (
	password = "testpassword"
)

func generatePassword() string {
	h := sha256.New()
	h.Write([]byte(password))
	return fmt.Sprintf("%x", h.Sum(nil))
}
