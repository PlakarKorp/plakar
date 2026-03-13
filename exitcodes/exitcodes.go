// Package exitcodes defines standardized exit codes for plakar CLI.
//
// Codes follow sysexits.h(3) conventions where applicable.
// See sysexits(3) for background on the standard exit code ranges.
package exitcodes

const (
	// Success indicates the command completed successfully.
	Success = 0

	// Failure is a general error not covered by a more specific code.
	Failure = 1

	// Usage indicates invalid command-line arguments or flags.
	// Corresponds to EX_USAGE from sysexits.h.
	Usage = 64

	// RepoNotFound indicates the repository could not be opened or located.
	// Corresponds to EX_NOINPUT from sysexits.h.
	RepoNotFound = 66

	// RepoIncompatible indicates a repository version mismatch.
	// Corresponds to EX_CONFIG from sysexits.h.
	RepoIncompatible = 78

	// AuthFailure indicates an authentication or decryption error
	// (wrong passphrase, missing keyfile, locked repository).
	// Corresponds to EX_NOPERM from sysexits.h.
	AuthFailure = 77

	// IntegrityFailure indicates a data integrity check failed
	// (corrupted chunks, Merkle tree mismatch).
	// Corresponds to EX_DATAERR from sysexits.h.
	IntegrityFailure = 65
)
