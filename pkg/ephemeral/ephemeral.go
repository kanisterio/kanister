package ephemeral

// Applier is the interface which applies a manipulation to the PodOption to be
// used to run ephemeral pdos.
type Applier interface {
	Apply(any)
}

// ApplierFunc is a function which implements the Applier interface and can be
// used to generically manipulate the PodOptions.
type ApplierFunc func(any)

func (f ApplierFunc) Apply(options any) { f(options) }

var (
	// Options holds the registered Appliers.
	Options ApplierList
)

// ApplierList is an array of registered Appliers which will be applied on
// a PodOption.
type ApplierList []Applier

// Apply calls the Applier::Apply method on all registered appliers.
func (l ApplierList) Apply(options any) {
	for _, applier := range l {
		applier.Apply(options)
	}
}

// Register adds the applier to the list of Appliers to be used when
// manipulating the PodOptions.
func (l *ApplierList) Register(applier Applier) {
	*l = append(*l, applier)
}

// Filterer is the interface which filters the use of registered appliers to
// only those PodOptions that match the filter criteria.
type Filterer interface {
	Filter(any) bool
}

// FiltererFunc is a function which implements the Filterer interface and can be
// used to generically filter PodOptions to manipulate using the ApplierList.
type FiltererFunc func(any) bool

func (f FiltererFunc) Filter(options any) bool {
	return f(options)
}

// PodNameFilter is a Filterer that filters based on the PodOptions.Name.
type PodNameFilter string

func (f PodNameFilter) Filter(options any) bool {
	if v, ok := options.(struct{ Name string }); ok {
		return string(f) == v.Name
	}

	return false
}

// Filter applies the Applier's if the Filterer criterion is met.
func Filter(filterer Filterer, appliers ...Applier) Applier {
	return ApplierFunc(func(options any) {
		if !filterer.Filter(options) {
			return
		}

		for _, applier := range appliers {
			applier.Apply(options)
		}
	})
}
