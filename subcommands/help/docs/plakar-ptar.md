PLAKAR-PTAR(1) - General Commands Manual

# NAME

**plakar-ptar** - generate a self-contained Kloset archive (.ptar)

# SYNOPSIS

**plakar&nbsp;ptar**
\[**-plaintext**]
\[**-overwrite**]
\[**-k**&nbsp;*location*]
**-o**&nbsp;*file.ptar*
\[*path&nbsp;...*]

# DESCRIPTION

The
**plakar ptar**
command creates a single portable archive
(a
'.ptar'
file) that bundles one or more existing Plakar repositories
("klosets")
and/or arbitrary filesystem paths into a self-contained package.
The resulting archive preserves repository metadata, snapshots and
data chunks, and is compressed and encrypted for secure transport or
off-site storage.

At least one data source must be supplied: either one or more
**-k** or **-kloset**
options naming remote or local kloset repositories, and/or one or more
*path*
arguments identifying files or directories to back up.
The destination archive name is mandatory and supplied with
**-o**.

Unless the
**-overwrite**
flag is given,
**plakar ptar**
refuses to replace an existing archive.

The options are as follows:

**-plaintext**

> Disable transparent encryption of the archive.
> If omitted,
> **plakar ptar**
> encrypts repository data using a key derived from the passphrase
> specified via
> `PLAKAR_PASSPHRASE`
> or prompted interactively.

**-overwrite**

> Overwrite an existing
> *.ptar*
> file at the destination path.

**-k** *location*, **-kloset** *location*

> Add a kloset repository to include in the archive.
> May be specified multiple times to bundle several repositories.

**-o** *file.ptar*

> Path of the archive to create.
> This option is required.

*path ...*

> Zero or more filesystem paths to back up directly into the archive.

# ENVIRONMENT

`PLAKAR_PASSPHRASE`

> Passphrase used to derive the encryption key when encryption is
> enabled.

# DIAGNOSTICS

The **plakar-ptar** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred (invalid arguments, existing archive without
> **-overwrite**,
> hashing algorithm unknown, repository access failure, I/O errors, etc.).

# SEE ALSO

plakar(1),
plakar-backup(1),
plakar-create(1)

Plakar - July 3, 2025
