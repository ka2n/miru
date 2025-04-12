package cli

// ErrorCode defines error types for CLI operations
type ErrorCode string

const (
	NoPackageSpecified  ErrorCode = "NoPackageSpecified"
	InvalidLanguageFlag ErrorCode = "InvalidLanguageFlag"
	InvalidLanguage     ErrorCode = "InvalidLanguage"
	InvalidArguments    ErrorCode = "InvalidArguments"
	UnsupportedLanguage ErrorCode = "UnsupportedLanguage"
	UnsupportedSource   ErrorCode = "UnsupportedSource"
)

func (c ErrorCode) ErrorCode() string {
	return string(c)
}
