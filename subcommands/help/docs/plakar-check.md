PLAKAR-CHECK(1) - General Commands Manual

# NAME

**plakar-check** - Check data integrity in a Plakar repository

# SYNOPSIS

**plakar&nbsp;check**
\[**-concurrency**&nbsp;*number*]
\[**-fast**]
\[**-no-verify**]
\[**-quiet**]
\[*snapshotID*:*path&nbsp;...*]

# DESCRIPTION

The
**plakar check**
command verifies the integrity of data in a Plakar repository.
It checks the given paths inside the snapshots for consistency and
validates file macs to ensure no corruption has occurred, or all
the data in the repository if no
*snapshotID*
or location flags is given.

In addition to the flags described below,
**plakar check**
supports the location flags documented in
plakar-query(7)
to precisely select snapshots.

The options are as follows:

**-concurrency** *number*

> Set the maximum number of parallel tasks for faster processing.
> Defaults to
> `8 * CPU count + 1`.

**-fast**

> Enable a faster check that skips mac verification.
> This option performs only structural validation without confirming
> data integrity.

**-no-verify**

> Disable signature verification.
> This option allows to proceed with checking snapshot integrity
> regardless of an invalid snapshot signature.

**-quiet**

> Suppress output to standard output, only logging errors and warnings.

# EXAMPLES

Perform a full integrity check on all snapshots:

	$ plakar check

Perform a fast check on specific paths of two snapshot:

	$ plakar check -fast abc123:/etc/passwd def456:/var/www

# DIAGNOSTICS

The **plakar-check** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully with no integrity issues found.

&gt;0

> An error occurred, such as corruption detected in a snapshot or
> failure to check data integrity.

# SEE ALSO

plakar(1),
plakar-query(7)

Plakar - September 10, 2025 - PLAKAR-CHECK(1)
