package restic

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	password = "testpassword"
)

func generatePassword() (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
