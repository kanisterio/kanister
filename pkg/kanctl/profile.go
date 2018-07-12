package kanctl

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/validate"
)

const (
	schemaValidation      = "Validate Profile schema"
	regionValidation      = "Validate bucket region specified in profile"
	readAccessValidation  = "Validate read access to bucket specified in profile"
	writeAccessValidation = "Validate write access to bucket specified in profile"
)

func performProfileValidation(p *params) error {
	ctx := context.Background()
	cli, crCli, err := initializeClients()
	if err != nil {
		return errors.Wrap(err, "could not initialize clients for validation")
	}
	prof, err := getProfileFromCmd(ctx, crCli, p)
	if err != nil {
		return err
	}

	if err := validate.ProfileSchema(prof); err != nil {
		printStage(schemaValidation, fail)
		return err
	}
	printStage(schemaValidation, pass)

	for _, d := range []string{regionValidation, readAccessValidation, writeAccessValidation} {
		if p.schemaValidationOnly {
			printStage(d, skip)
			continue
		}
		switch d {
		case regionValidation:
			err = validate.ProfileBucket(ctx, prof)
		case readAccessValidation:
			err = validate.ReadAccess(ctx, prof, cli)
		case writeAccessValidation:
			err = validate.WriteAccess(ctx, prof, cli)
		}
		if err != nil {
			printStage(d, fail)
			return err
		}
		printStage(d, pass)
	}
	printStage(fmt.Sprintf("All checks passed.. %s\n", pass), "")
	return nil
}

func getProfileFromCmd(ctx context.Context, crCli versioned.Interface, p *params) (*v1alpha1.Profile, error) {
	if p.name != "" {
		return crCli.CrV1alpha1().Profiles(p.namespace).Get(p.name, v1.GetOptions{})
	}
	return getProfileFromFile(ctx, p.filename)
}

func getProfileFromFile(ctx context.Context, filename string) (*v1alpha1.Profile, error) {
	var f *os.File
	var err error

	if filename == "-" {
		f = os.Stdin
	} else {
		f, err = os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()
	}
	d := yaml.NewYAMLOrJSONDecoder(f, 4096)
	prof := &v1alpha1.Profile{}
	err = d.Decode(prof)
	if err != nil {
		return nil, err
	}
	return prof, nil
}
