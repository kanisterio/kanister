package kando

import (
	"context"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/kopia/snapshot"
)

type Profile struct {
	Command   *cobra.Command
	Arguments []string
}

func (p *Profile) Push() error {
	path := pathFlag(p.Command)
	ctx := context.Background()
	outputName := outputNameFlag(p.Command)
	profile, err := unmarshalProfileFlag(p.Command)
	if err != nil {
		return err
	}
	if profile.Location.Type == crv1alpha1.LocationTypeKopia {
		if err = connectToKopiaServer(ctx, profile); err != nil {
			return err
		}
		return kopiaLocationPush(ctx, path, outputName, p.Arguments[0], profile.Credential.KopiaServerSecret.Password)
	}
	source, err := sourceReader(p.Arguments[0])
	if err != nil {
		return err
	}
	return locationPush(ctx, profile, path, source)

}

func (p *Profile) Pull() error {
	path := pathFlag(p.Command)
	ctx := context.Background()
	profile, err := unmarshalProfileFlag(p.Command)
	if err != nil {
		return err
	}
	if profile.Location.Type == crv1alpha1.LocationTypeKopia {
		snapJSON := kopiaSnapshotFlag(p.Command)
		if snapJSON == "" {
			return errors.New("kopia snapshot information is required to pull data using kopia")
		}
		kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(snapJSON)
		if err != nil {
			return err
		}
		if err = connectToKopiaServer(ctx, profile); err != nil {
			return err
		}
		return kopiaLocationPull(ctx, kopiaSnap.ID, path, p.Arguments[0], profile.Credential.KopiaServerSecret.Password)
	}
	target, err := targetWriter(p.Arguments[0])
	if err != nil {
		return err
	}
	return locationPull(ctx, profile, path, target)

}

func (p *Profile) Delete() error {
	p.Command.SilenceUsage = true
	path := pathFlag(p.Command)
	ctx := context.Background()
	profile, err := unmarshalProfileFlag(p.Command)
	if err != nil {
		return err
	}
	if profile.Location.Type == crv1alpha1.LocationTypeKopia {
		snapJSON := kopiaSnapshotFlag(p.Command)
		if snapJSON == "" {
			return errors.New("kopia snapshot information is required to delete data using kopia")
		}
		kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(snapJSON)
		if err != nil {
			return err
		}
		if err = connectToKopiaServer(ctx, profile); err != nil {
			return err
		}
		return kopiaLocationDelete(ctx, kopiaSnap.ID, path, profile.Credential.KopiaServerSecret.Password)
	}
	return locationDelete(ctx, profile, path)
}
