PLAKAR-CP(1) - General Commands Manual

# NAME

**plakar-cp** - Copy files between two locations

# SYNOPSIS

**plakar&nbsp;cp**
\[**-quiet**]
*source*
*destination*

# DESCRIPTION

The
**plakar cp**
command copies files from
*source*
to
*destination*
without going through a Kloset store.
Both endpoints can be any location supported by the configured
connectors, which makes it possible to copy between, say, a local
directory and an S3 bucket.

*source*
and
*destination*
are locations, but a name prefixed with
'@'
refers to a source or a destination declared in the configuration, refer
to
plakar-source(1)
and
plakar-destination(1).

The copy is rooted at the last component of the source, the way
cp(1)
copies a directory: copying
*/var/www*
to
*/srv/backup*
fills
*/srv/backup/www*
and not
*/srv/backup/var/www*.

Every copied file is listed on the standard output as the destination
acknowledges it.

The options are as follows:

**-quiet**

> Do not list the copied files.

# EXIT STATUS

The **plakar-cp** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

Failing to copy an entry is not fatal: the error is reported and the copy
carries on with the remaining files, but
**plakar cp**
exits with a non-zero status.

# EXAMPLES

Copy a local directory to another one:

	$ plakar cp /var/www /srv/backup

Copy a local directory to a configured destination:

	$ plakar destination add mybucket location=s3://bucket/path
	$ plakar cp /var/www @mybucket

# SEE ALSO

plakar(1),
plakar-backup(1),
plakar-destination(1),
plakar-restore(1),
plakar-source(1)

Plakar - August 12, 2026 - PLAKAR-CP(1)
