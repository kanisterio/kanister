package datamover

import (
	"github.com/spf13/cobra"
)

type DataMover interface {
	// Pull is used to restore the data from object storage
	// using the preferred data-mover
	Pull(sourcePath, destinationPath string) error
	// Push is used to backup the data to object storage
	// using the preferred data-mover
	Push(sourcePath, destinationPath string) error
	// Delete is used to delete the data from object storage
	// using the preferred data-mover
	Delete(destinationPath string) error
}

// NewDataMover creates an instance of DataMover Interface and returns
// the preferred DataMover as per the arguments passed in kando command
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
