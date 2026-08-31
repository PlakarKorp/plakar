PLAKAR-PTAR(1) - General Commands Manual

# NAME

**plakar-ptar** - generate a self-contained Kloset archive (.ptar)

# SYNOPSIS

**plakar&nbsp;ptar**
\[**-compression**&nbsp;*algorithm*]
\[**-no-compression**]
\[**-plaintext**]
\[**-overwrite**]
\[**-ignore**&nbsp;*pattern*]
\[**-ignore-file**&nbsp;*file*]
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

In addition to the flags described below,
**plakar ptar**
supports the location flags documented in
plakar-query(7)
to precisely select which snapshots of the
**-k**
klosets are bundled.
Without any of them every snapshot is included.
They do not affect the
*path*
arguments, which are always backed up in full.

The options are as follows:

**-compression**&nbsp;*algorithm*

> Select the algorithm used for transparent compression.
> Supported values are
> **LZ4**,
> **GZIP**
> and
> **ZSTD**,
> and the name is matched case-insensitively.
> Defaults to
> **LZ4**.

**-no-compression**

> Disable transparent compression of the archive.
> Mutually exclusive with
> **-compression**.

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

**-ignore** *pattern*

> Exclude files matching a gitignore-style pattern while backing up filesystem
> paths into the archive.
> May be specified multiple times.

**-ignore-file** *file*

> Read newline-separated gitignore-style patterns from
> *file*.
> Blank lines and comments are ignored.
> May be specified multiple times.

**-k** *location*, **-kloset** *location*

> Add a kloset repository to include in the archive.
> May be specified multiple times to bundle several repositories.
> The location flags apply to each of them.

**-o** *file.ptar*

> Path of the archive to create.
> This option is required.

*path ...*

> Zero or more filesystem paths to back up directly into the archive.

# ENVIRONMENT

`PLAKAR_PASSPHRASE`

> Passphrase used to derive the encryption key when encryption is
> enabled.

# EXIT STATUS

The **plakar-ptar** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1),
plakar-backup(1),
plakar-create(1)

Plakar - May 5, 2026 - PLAKAR-PTAR(1)
