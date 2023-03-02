package repositoryserver

type RepositoryServerSecrets interface {
	ValidateSecret() error
}
