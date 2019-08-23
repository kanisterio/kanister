package format

import (
	"bufio"
	"io"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

func Log(podName string, containerName string, output string) {
	if output != "" {
		logs := regexp.MustCompile("[\r\n]").Split(output, -1)
		for _, l := range logs {
			info(podName, containerName, l)
		}
	}
}

func LogStream(podName string, containerName string, output io.ReadCloser) chan string {
	logCh := make(chan string)
	s := bufio.NewScanner(output)
	go func() {
		defer close(logCh)
		for s.Scan() {
			l := s.Text()
			info(podName, containerName, l)
			logCh <- l
		}
		if err := s.Err(); err != nil {
			log.Error("Pod: ", podName, " Container: ", containerName, " Failed to stream log from pod: ", err.Error())
		}
	}()
	return logCh
}

func info(podName string, containerName string, l string) {
	if strings.TrimSpace(l) != "" {
		log.Info("Pod: ", podName, " Container: ", containerName, " Out: ", l)
	}
}
