package sourceimpl

type ErrorCode string

const (
	ErrInvalidPackagePath ErrorCode = "InvalidPackagePath"

	// ErrRepositoryNotFound represents errors when repository information cannot be found
	ErrRepositoryNotFound ErrorCode = "RepositoryNotFound"
)
