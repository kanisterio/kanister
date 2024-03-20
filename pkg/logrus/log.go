package logrus

import "github.com/sirupsen/logrus"

// The logging library in package `pkg/log` tries to figure out the cluster ID
// by getting the default namespace from K8S cluster. People who are using a
// kanister utility that doesn't necessarily need to communicate with K8S,
// would get confused by why that utility (`kando` for example) is trying to
// communicate with K8S.
// Thats the reason from those utilities, instead of using our loggging library
// we can directly use logrus to log something.
func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetReportCaller(true)
}
