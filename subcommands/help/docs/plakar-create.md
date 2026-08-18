PLAKAR-CREATE(1) - General Commands Manual

# NAME

**plakar-create** - Create a new Plakar repository

# SYNOPSIS

**plakar&nbsp;create**
\[**-compression**&nbsp;*algorithm*]
\[**-no-compression**]
\[**-plaintext**]

# DESCRIPTION

The
**plakar create**
command creates a new Plakar repository at the specified path which defaults to
*~/.plakar*.

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
> The algorithm is recorded in the repository configuration when it is created and
> cannot be changed afterwards.

**-no-compression**

> Disable transparent compression for the repository.
> Mutually exclusive with
> **-compression**.

**-plaintext**

> Disable transparent encryption for the repository.
> If specified, the repository will not use encryption.

# ENVIRONMENT

`PLAKAR_PASSPHRASE`

> Repository encryption password.

# EXIT STATUS

The **plakar-create** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1),
plakar-backup(1)

Plakar - May 5, 2026 - PLAKAR-CREATE(1)
