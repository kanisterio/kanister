package kando

import (
	"context"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kopia/snapshot"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type Location interface {
	Push() error
	Pull() error
	Delete() error
}

type Command struct {
	Subcommand *cobra.Command
	Arguments  []string
}

func (c *Command) Push() error {
	profile, repositoryServer := profileAndRepositoryServerFlagFromCommand(c.Subcommand)
	path := pathFlag(c.Subcommand)
	ctx := context.Background()
	outputName := outputNameFlag(c.Subcommand)

	if profile != "" {
		p, err := unmarshalProfileFlag(c.Subcommand)
		if err != nil {
			return err
		}
		if p.Location.Type == crv1alpha1.LocationTypeKopia {
			if err = connectToKopiaServer(ctx, p); err != nil {
				return err
			}
			return kopiaLocationPush(ctx, path, outputName, c.Arguments[0], p.Credential.KopiaServerSecret.Password)
		}
		source, err := sourceReader(c.Arguments[0])
		if err != nil {
			return err
		}
		return locationPush(ctx, p, path, source)
	}

	if repositoryServer != "" {
		rs, err := unmarshalRepositoryServerFlag(c.Subcommand)
		if err != nil {
			return err
		}
		err, password := connectToKopiaRepositoryServer(ctx, rs)
		if err != nil {
			return err
		}
		return kopiaLocationPush(ctx, path, outputName, c.Arguments[0], password)
	}

	if profile != "" && repositoryServer != "" {
		return errors.New("Please Provide either --profile / --kopia-repo-server")
	}

	return nil
}

func (c *Command) Pull() error {
	profile, repositoryServer := profileAndRepositoryServerFlagFromCommand(c.Subcommand)
	path := pathFlag(c.Subcommand)
	ctx := context.Background()

	if profile != "" {
		p, err := unmarshalProfileFlag(c.Subcommand)
		if err != nil {
			return err
		}
		if p.Location.Type == crv1alpha1.LocationTypeKopia {
			snapJSON := kopiaSnapshotFlag(c.Subcommand)
			if snapJSON == "" {
				return errors.New("kopia snapshot information is required to pull data using kopia")
			}
			kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(snapJSON)
			if err != nil {
				return err
			}
			if err = connectToKopiaServer(ctx, p); err != nil {
				return err
			}
			return kopiaLocationPull(ctx, kopiaSnap.ID, path, c.Arguments[0], p.Credential.KopiaServerSecret.Password)
		}
		target, err := targetWriter(c.Arguments[0])
		if err != nil {
			return err
		}
		return locationPull(ctx, p, path, target)
	}

	if repositoryServer != "" {
		rs, err := unmarshalRepositoryServerFlag(c.Subcommand)
		if err != nil {
			return err
		}
		snapJSON := kopiaSnapshotFlag(c.Subcommand)
		if snapJSON == "" {
			return errors.New("kopia snapshot information is required to pull data using kopia")
		}
		kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(snapJSON)
		if err != nil {
			return err
		}
		err, password := connectToKopiaRepositoryServer(ctx, rs)
		if err != nil {
			return err
		}
		return kopiaLocationPull(ctx, kopiaSnap.ID, path, c.Arguments[0], password)
	}

	if profile != "" && repositoryServer != "" {
		return errors.New("Please Provide either --profile / --kopia-repo-server")
	}

	return nil
}

func (c *Command) Delete() error {
	profile, repositoryServer := profileAndRepositoryServerFlagFromCommand(c.Subcommand)
	c.Subcommand.SilenceUsage = true
	path := pathFlag(c.Subcommand)
	ctx := context.Background()

	if profile != "" {
		p, err := unmarshalProfileFlag(c.Subcommand)
		if err != nil {
			return err
		}
		if p.Location.Type == crv1alpha1.LocationTypeKopia {
			snapJSON := kopiaSnapshotFlag(c.Subcommand)
			if snapJSON == "" {
				return errors.New("kopia snapshot information is required to delete data using kopia")
			}
			kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(snapJSON)
			if err != nil {
				return err
			}
			if err = connectToKopiaServer(ctx, p); err != nil {
				return err
			}
			return kopiaLocationDelete(ctx, kopiaSnap.ID, path, p.Credential.KopiaServerSecret.Password)
		}
		return locationDelete(ctx, p, path)
	}

	if repositoryServer != "" {
		rs, err := unmarshalRepositoryServerFlag(c.Subcommand)
		if err != nil {
			return err
		}
		snapJSON := kopiaSnapshotFlag(c.Subcommand)
		if snapJSON == "" {
			return errors.New("kopia snapshot information is required to pull data using kopia")
		}
		kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(snapJSON)
		if err != nil {
			return err
		}
		err, password := connectToKopiaRepositoryServer(ctx, rs)
		if err != nil {
			return err
		}
		return kopiaLocationDelete(ctx, kopiaSnap.ID, path, password)
	}

	if profile != "" && repositoryServer != "" {
		return errors.New("Please Provide either --profile / --kopia-repo-server")
	}

	return nil
}

func profileAndRepositoryServerFlagFromCommand(c *cobra.Command) (string, string) {
	return c.Flag(profileFlagName).Value.String(), c.Flag(repositoryServerFlagName).Value.String()
}
