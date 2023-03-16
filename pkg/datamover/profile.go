package datamover

import (
	"context"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
)

type Profile struct {
	OutputName string
	Profile    *param.Profile
	SnapJSON   string
}

func (p *Profile) Pull(sourcePath, destinationPath string) error {
	ctx := context.Background()
	if p.Profile.Location.Type == crv1alpha1.LocationTypeKopia {
		if p.SnapJSON == "" {
			return errors.New("kopia snapshot information is required to pull data using kopia")
		}
		kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(p.SnapJSON)
		if err != nil {
			return err
		}
		if err = connectToKopiaServer(ctx, p.Profile); err != nil {
			return err
		}
		return kopiaLocationPull(ctx, kopiaSnap.ID, destinationPath, sourcePath, p.Profile.Credential.KopiaServerSecret.Password)
	}
	target, err := targetWriter(sourcePath)
	if err != nil {
		return err
	}
	return locationPull(ctx, p.Profile, destinationPath, target)
}

func (p *Profile) Push(sourcePath, destinationPath string) error {
	ctx := context.Background()
	if p.Profile.Location.Type == crv1alpha1.LocationTypeKopia {
		if err := connectToKopiaServer(ctx, p.Profile); err != nil {
			return err
		}
		return kopiaLocationPush(ctx, destinationPath, p.OutputName, sourcePath, p.Profile.Credential.KopiaServerSecret.Password)
	}
	source, err := sourceReader(sourcePath)
	if err != nil {
		return err
	}
	return locationPush(ctx, p.Profile, destinationPath, source)
}

func (p *Profile) Delete(destinationPath string) error {
	ctx := context.Background()
	if p.Profile.Location.Type == crv1alpha1.LocationTypeKopia {
		if p.SnapJSON == "" {
			return errors.New("kopia snapshot information is required to pull data using kopia")
		}
		kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(p.SnapJSON)
		if err != nil {
			return err
		}
		if err = connectToKopiaServer(ctx, p.Profile); err != nil {
			return err
		}
		return kopiaLocationDelete(ctx, kopiaSnap.ID, destinationPath, p.Profile.Credential.KopiaServerSecret.Password)
	}
	return locationDelete(ctx, p.Profile, destinationPath)
}
