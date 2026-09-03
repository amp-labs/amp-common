package feature

// Dimensions that a consumer of this package would define for its own domain.
// They live in a test file so that the library itself stays domain-neutral.
const (
	dimUser         Dimension = "user"
	dimInstallation Dimension = "installation"
)
