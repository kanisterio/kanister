package v1alpha1

// DeepCopyInto handles BlueprintPhase deep copies, copying the receiver, writing into out. in must be non-nil.
// The auto-generated function does not handle the map[string]interface{} type
func (in *BlueprintPhase) DeepCopyInto(out *BlueprintPhase) {
	*out = *in
	// TODO: Handle 'Args'
}

// DeepCopyInto handles the Phase deep copies, copying the receiver, writing into out. in must be non-nil.
// This is a workaround to handle the map[string]interface{} output type
func (in *Phase) DeepCopyInto(out *Phase) {
	*out = *in
	// TODO: Handle 'Output' map[string]interface{}
}

// DeepCopyInto handles JSONMap deep copies, copying the receiver, writing into out. in must be non-nil.
// The auto-generated function does not handle the map[string]interface{} type
func (in *JSONMap) DeepCopyInto(out *JSONMap) {
	*out = *in
}
