package datamover

import (
	"context"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
)

type RepositoryServer struct {
	OutputName       string
	RepositoryServer *param.RepositoryServer
	SnapJSON         string
}

func (rs *RepositoryServer) Pull(sourcePath, destinationPath string) error {
	ctx := context.Background()
	if rs.SnapJSON == "" {
		return errors.New("kopia snapshot information is required to pull data using kopia")
	}
	kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(rs.SnapJSON)
	if err != nil {
		return err
	}
	password, err := connectToKopiaRepositoryServer(ctx, rs.RepositoryServer)
	if err != nil {
		return err
	}
	return kopiaLocationPull(ctx, kopiaSnap.ID, destinationPath, sourcePath, password)
}

func (rs *RepositoryServer) Push(sourcePath, destinationPath string) error {
	ctx := context.Background()
	password, err := connectToKopiaRepositoryServer(ctx, rs.RepositoryServer)
	if err != nil {
		return err
	}
	return kopiaLocationPush(ctx, destinationPath, rs.OutputName, sourcePath, password)
}

func (rs *RepositoryServer) Delete(destinationPath string) error {
	ctx := context.Background()
	if rs.SnapJSON == "" {
		return errors.New("kopia snapshot information is required to pull data using kopia")
	}
	kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(rs.SnapJSON)
	if err != nil {
		return err
	}
	password, err := connectToKopiaRepositoryServer(ctx, rs.RepositoryServer)
	if err != nil {
		return err
	}
	return kopiaLocationDelete(ctx, kopiaSnap.ID, destinationPath, password)
}
