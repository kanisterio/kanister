package datamover

import (
	"github.com/spf13/cobra"
)

type DataMover interface {
	Pull(sourcePath, destinationPath string) error
	Push(sourcePath, destinationPath string) error
	Delete(destinationPath string) error
}

func NewDataMover(c *cobra.Command) DataMover {
	datamover := checkDataMover(c)
	switch datamover {
	case profileFlagName:
		outputName := outputNameFlag(c)
		profileRef, err := unmarshalProfileFlag(c)
		if err != nil {
			return nil
		}
		snapJSON := kopiaSnapshotFlag(c)
		return &Profile{
			OutputName: outputName,
			Profile:    profileRef,
			SnapJSON:   snapJSON,
		}
	case repositoryServerFlagName:
		outputName := outputNameFlag(c)
		repositoryServerRef, err := unmarshalRepositoryServerFlag(c)
		if err != nil {
			return nil
		}
		snapJSON := kopiaSnapshotFlag(c)
		return &RepositoryServer{
			OutputName:       outputName,
			RepositoryServer: repositoryServerRef,
			SnapJSON:         snapJSON,
		}
	default:
		return nil
	}
}
