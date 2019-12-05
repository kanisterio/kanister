// Package servicelog does not export any symbols. Its main functionality is to
// initialize the log options for a service. A server program should import
// this package for side effects only as follows:
//
//   import _ "kio/servicelog" // Initializes log
//
package servicelog

import (
	"github.com/kanisterio/kanister/pkg/log"
)

func init() {
	err := log.SetOutput(log.FluentbitSink)
	if err != nil {
		log.Error().WithError(err).Print("Unable to set Fluentbit as the log sink.")
	}
}
