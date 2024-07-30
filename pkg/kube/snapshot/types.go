package snapshot

// ObjectMeta is metadata of snapshot which is configurable.
type ObjectMeta struct {
	// Name must be unique within a namespace. Is required when creating resources.
	Name string `json:"name,omitempty"`
	// Namespace defines the space within which a resource is created.
	Namespace string `json:"namespace,omitempty"`
	// Labels are set to group a resource. This can be used to filter certain resource.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations is an unstructured key value map stored. This can be used to retrieve arbitrary metadata.
	Annotations map[string]string `json:"annotations,omitempty"`
}
