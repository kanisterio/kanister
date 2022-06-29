package v1alpha1

const (
	// ProfileTypeExport captures enum value "Export"
	ProfileTypeExport string = "Export"
	// ProfileTypeImport captures enum value "Import"
	ProfileTypeImport string = "Import"
	// ProfileTypeKanister captures enum value "Kanister"
	ProfileTypeKanister string = "Kanister"
	// ProfileTypeInfra captures enum value "Infra"
	ProfileTypeInfra string = "Infra"
	// ProfileTypeLocation captures enum value "Location"
	ProfileTypeLocation string = "Location"

	// LocationObjectStorePathTypeDirectory captures enum value "Directory"
	LocationObjectStorePathTypeDirectory string = "Directory"
	// LocationObjectStorePathTypeObject captures enum value "Object"
	LocationObjectStorePathTypeObject string = "Object"

	// LocationTypeFileStore captures enum value "FileStore"
	LocationTypeFileStore LocationType = "FileStore"
	// LocationTypeObjectStore captures enum value "ObjectStore"
	LocationTypeObjectStore LocationType = "ObjectStore"
	// LocationTypeVBR captures enum value "VBR"
	LocationTypeVBR LocationType = "VBR"

	// SecretTypeAwsAccessKey captures enum value "AwsAccessKey"
	SecretTypeAwsAccessKey SecretType = "AwsAccessKey"
	// SecretTypeGcpServiceAccountKey captures enum value "GcpServiceAccountKey"
	SecretTypeGcpServiceAccountKey SecretType = "GcpServiceAccountKey"
	// SecretTypeAzStorageAccount captures enum value "AzStorageAccount"
	SecretTypeAzStorageAccount SecretType = "AzStorageAccount"
	// SecretTypeAPIKey captures enum value "ApiKey"
	SecretTypeAPIKey SecretType = "ApiKey"
	// SecretTypeOpenStackAccount captures enum value "OpenStackAccount"
	SecretTypeOpenStackAccount SecretType = "OpenStackAccount"
	// SecretTypeCephKeyring captures enum value "CephKeyring"
	SecretTypeCephKeyring SecretType = "CephKeyring"
	// SecretTypePortworxKey captures enum value "PortworxKey"
	SecretTypePortworxKey SecretType = "PortworxKey"
	// SecretTypeVSphereKey captures enum value "VSphereKey"
	SecretTypeVSphereKey SecretType = "VSphereKey"
	// SecretTypeVBRKey captures enum value "VBRKey"
	SecretTypeVBRKey SecretType = "VBRKey"

	// DefaultKanisterProfile is the defaut kanister profile name
	DefaultKanisterProfile = "kanister-profile"

	// DefaultInfraNameGCP is the default GCP infrastructure profile name
	DefaultInfraNameGCP string = "default-gcp-profile"
	// DefaultInfraNameAWS is the default AWS infrastructure profile name
	DefaultInfraNameAWS string = "default-aws-profile"
)
