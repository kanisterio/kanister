package kando

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/kopia/snapshot"
)

type RepositoryServer struct {
	Command   *cobra.Command
	Arguments []string
}

func (rs *RepositoryServer) Push() error {
	path := pathFlag(rs.Command)
	ctx := context.Background()
	outputName := outputNameFlag(rs.Command)
	repositoryServer, err := unmarshalRepositoryServerFlag(rs.Command)
	if err != nil {
		return err
	}
	err, password := connectToKopiaRepositoryServer(ctx, repositoryServer)
	if err != nil {
		return err
	}
	return kopiaLocationPush(ctx, path, outputName, rs.Arguments[0], password)
}

func (rs *RepositoryServer) Pull() error {
	path := pathFlag(rs.Command)
	ctx := context.Background()
	repositoryServer, err := unmarshalRepositoryServerFlag(rs.Command)
	if err != nil {
		return err
	}
	snapJSON := kopiaSnapshotFlag(rs.Command)
	if snapJSON == "" {
		return errors.New("kopia snapshot information is required to pull data using kopia")
	}
	kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(snapJSON)
	if err != nil {
		return err
	}
	err, password := connectToKopiaRepositoryServer(ctx, repositoryServer)
	if err != nil {
		return err
	}
	return kopiaLocationPull(ctx, kopiaSnap.ID, path, rs.Arguments[0], password)
}

func (rs *RepositoryServer) Delete() error {
	rs.Command.SilenceUsage = true
	path := pathFlag(rs.Command)
	ctx := context.Background()
	repositoryServer, err := unmarshalRepositoryServerFlag(rs.Command)
	if err != nil {
		return err
	}
	snapJSON := kopiaSnapshotFlag(rs.Command)
	if snapJSON == "" {
		return errors.New("kopia snapshot information is required to pull data using kopia")
	}
	kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(snapJSON)
	if err != nil {
		return err
	}
	err, password := connectToKopiaRepositoryServer(ctx, repositoryServer)
	if err != nil {
		return err
	}
	return kopiaLocationDelete(ctx, kopiaSnap.ID, path, password)
}
