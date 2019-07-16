package blockstorage

// Type is the type of storage supported
type Type string

const (
	// TypeAD captures enum value "AD"
	TypeAD Type = "AD"
	// TypeEBS captures enum value "EBS"
	TypeEBS Type = "EBS"
	// TypeGPD captures enum value "GPD"
	TypeGPD Type = "GPD"
	// TypeCinder captures enum value "Cinder"
	TypeCinder Type = "Cinder"
	// TypeGeneric captures enum value "Generic"
	TypeGeneric Type = "Generic"
	// TypeCeph captures enum value "Ceph"
	TypeCeph Type = "Ceph"
	// TypeSoftlayerBlock captures enum value "SoftlayerBlock"
	TypeSoftlayerBlock Type = "SoftlayerBlock"
	// TypeSoftlayerFile captures enum value "SoftlayerFile"
	TypeSoftlayerFile Type = "SoftlayerFile"
	// TypeEFS captures enum value "EFS"
	TypeEFS Type = "EFS"
)
