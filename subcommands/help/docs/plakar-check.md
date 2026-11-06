PLAKAR-CHECK(1) - General Commands Manual

# NAME

**plakar-check** - Check data integrity in a Plakar repository

# SYNOPSIS

**plakar&nbsp;check**
\[**-fast**]
\[**-no-verify**]
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

**-fast**

> Enable a faster check that skips mac verification.
> This option performs only structural validation without confirming
> data integrity.

**-no-verify**

> Disable signature verification.
> This option allows to proceed with checking snapshot integrity
> regardless of an invalid snapshot signature.

# EXIT STATUS

The **plakar-check** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Perform a full integrity check on all snapshots:

	$ plakar check

Perform a fast check on specific paths of two snapshot:

	$ plakar check -fast abc123:/etc/passwd def456:/var/www

# SEE ALSO

plakar(1),
plakar-query(7)

Plakar - May 5, 2025
