package format

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

func Log(podName string, containerName string, output string) {
	if output != "" {
		logs := regexp.MustCompile("[\r\n]").Split(output, -1)
		for _, l := range logs {
			if strings.TrimSpace(l) != "" {
				log.Info("Pod: ", podName, " Container: ", containerName, " Out: ", l)
			}
		}
	}
}
