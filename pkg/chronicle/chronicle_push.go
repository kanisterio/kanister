package chronicle

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/kanisterio/kanister/pkg/envdir"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

type PushParams struct {
	ProfilePath  string
	ArtifactPath string
	Frequency    time.Duration
	EnvDir       string
	Command      []string
}

func (p PushParams) Validate() error {
	return nil
}

func Push(p PushParams) error {
	log.Infof("%#v", p)
	ctx := setupSignalHandler(context.Background())
	var i int
	for {
		start := time.Now().UTC()

		if err := push(ctx, p, i); err != nil {
			return err
		}

		end := time.Now().UTC()
		sleep := p.Frequency - end.Sub(start)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(sleep):
		}
		i++
	}
}

func setupSignalHandler(ctx context.Context) context.Context {
	var can context.CancelFunc
	ctx, can = context.WithCancel(ctx)
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Infof("Shutting down process")
		can()
		<-c
		log.Infof("Killing process")
		os.Exit(1)
	}()
	return ctx
}

func push(ctx context.Context, p PushParams, ord int) error {
	// Read profile.
	prof, ok, err := readProfile(p.ProfilePath)
	if !ok || err != nil {
		return errors.Wrap(err, "")
	}

	// Get envdir values if set.
	var env []string
	if p.EnvDir != "" {
		var err error
		envdir.EnvDir(p.EnvDir)
		if err != nil {
			return err
		}
	}

	// Chronicle command w/ piped output.
	cmd := exec.CommandContext(ctx, "sh", "-c", strings.Join(p.Command, " "))
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "Failed to open command pipe")
	}
	cmd.Stderr = os.Stderr
	cur := fmt.Sprintf("%s-%d", p.ArtifactPath, ord)
	// Write data to object store
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "Failed to start chronicle pipe command")
	}
	if err := location.Write(ctx, out, prof, cur); err != nil {
		return errors.Wrap(err, "Failed to write command output to object storage")
	}
	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "Chronicle pipe command failed")
	}

	// Write manifest pointing to new data
	man := strings.NewReader(cur)
	if err := location.Write(ctx, man, prof, p.ArtifactPath); err != nil {
		return errors.Wrap(err, "Failed to write command output to object storage")
	}
	// Delete old data
	prev := fmt.Sprintf("%s-%d", p.ArtifactPath, ord-1)
	location.Delete(ctx, prof, prev)
	return nil
}

func readProfile(path string) (p param.Profile, ok bool, err error) {
	var buf []byte
	buf, err = ioutil.ReadFile(path)
	switch {
	case os.IsNotExist(err):
		ok = true
		err = nil
		return
	case err != nil:
		err = errors.Wrap(err, "Failed to read profile")
		return
	}
	if err = json.Unmarshal(buf, &p); err != nil {
		err = errors.Wrap(err, "Failed to unmarshal profile")
	} else {
		ok = true
	}
	return
}

func writeProfile(path string, p param.Profile) error {
	buf, err := json.Marshal(p)
	if err != nil {
		return errors.Wrap(err, "Failed to write profile")
	}
	return ioutil.WriteFile(path, buf, os.ModePerm)
}
