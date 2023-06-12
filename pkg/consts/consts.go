package consts

const (
	ActionsetNameKey = "ActionSet"
	PodNameKey       = "Pod"
	ContainerNameKey = "Container"
	PhaseNameKey     = "Phase"
	LogKindKey       = "LogKind"
	LogKindDatapath  = "datapath"

	GoogleCloudCredsFilePath = "/tmp/creds.txt"
	LabelKeyCreatedBy        = "createdBy"
	LabelValueKanister       = "kanister"
	LabelPrefix              = "kanister.io/"
)

// These names are used to query ActionSet API objects.
const (
	ActionSetResourceName       = "actionset"
	ActionSetResourceNamePlural = "actionsets"
	BlueprintResourceName       = "blueprint"
	BlueprintResourceNamePlural = "blueprints"
	ProfileResourceName         = "profile"
	ProfileResourceNamePlural   = "profiles"
)

// These consts are used to query Repository server API objects
const RepositoryServerResourceName = "repositoryserver"
const RepositoryServerResourceNamePlural = "repositoryservers"

const LatestKanisterToolsImage = "ghcr.io/kanisterio/kanister-tools:v9.99.9-dev"
const KanisterToolsImage = "ghcr.io/kanisterio/kanister-tools:0.92.0"
