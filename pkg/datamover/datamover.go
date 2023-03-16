package datamover

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
