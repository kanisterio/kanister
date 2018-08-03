package kanctl

import (
	"math/rand"
	"time"
)

const characters = "abcdefghijklmnopqrstuvwxyz0123456789"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randString(length int) string {
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		bytes[i] = byte(characters[rand.Intn(len(characters))])
	}
	return string(bytes)
}
