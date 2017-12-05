// Package main for a kanister operator
package main

import (
	"github.com/kanisterio/kanister/pkg/kanctl"
)

func init() {
	// We silence all non-fatal log messages.
	//logrus.SetLevel(logrus.ErrorLevel)
}

func main() {
	kanctl.Execute()
}
