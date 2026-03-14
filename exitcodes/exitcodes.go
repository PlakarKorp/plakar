// Package exitcodes defines standardized exit codes for plakar CLI.
//
// These codes allow automation tools and CI/CD pipelines to distinguish
// between different failure modes without parsing stderr output.
package exitcodes

const (
	// Success indicates the command completed successfully.
	Success = 0

	// Failure is a general error not covered by a more specific code.
	Failure = 1

	// Usage indicates invalid command-line arguments or flags.
	Usage = 2

	// RepoNotFound indicates the repository could not be opened or located.
	RepoNotFound = 10

	// RepoIncompatible indicates a repository version mismatch.
	RepoIncompatible = 11

	// AuthFailure indicates an authentication or decryption error
	// (wrong passphrase, missing keyfile, locked repository).
	AuthFailure = 12

	// IntegrityFailure indicates a data integrity check failed
	// (corrupted chunks, Merkle tree mismatch).
	IntegrityFailure = 13
)
