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

const LatestKanisterToolsImage = "gcr.io/kanisterio/kanister-tools:aaron_tag_k3"
const KanisterToolsImage = "gcr.io/kanisterio/kanister-tools:aaron_tag_k3"
