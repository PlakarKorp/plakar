PLAKAR-MOUNT(1) - General Commands Manual

# NAME

**plakar-mount** - Mount Plakar snapshots as read-only filesystem

# SYNOPSIS

**plakar&nbsp;mount**
\[**-allow-others**]
\[**-no-auth**]
\[**-to**&nbsp;*mountpoint*]
\[**-token**&nbsp;*string*]
\[*snapshotID*]

# DESCRIPTION

The
**plakar mount**
command mounts a Plakar repository snapshot as a read-only filesystem
at the specified
*mountpoint*.
This allows users to access snapshot contents as if they were part of
the local file system, providing easy browsing and retrieval of files
without needing to explicitly restore them.
This command may not work on all Operating Systems.

In addition to the flags described below,
**plakar mount**
supports the location flags documented in
plakar-query(7)
to precisely select snapshots.

The options are as follows:

**-allow-others**

> Allow other users to access the mounted filesystem.

**-no-auth**

> Serve an HTTP mount without requiring a token.
> The files served are decrypted, so anyone who can reach the address can
> read every snapshot.

**-to** *mountpoint*

> Specify the mount location.
> The
> *mountpoint*
> can either be:

> *	A directory path for FUSE mounts

> *	An HTTP address including port for remote mounting (e.g.,
> 	'`http://hostname:8080`')

> If not specified, mount will attempt a FUSE mount in the working directory with
> a random subdirectory name.

**-token** *string*

> Require clients of an HTTP mount to provide the given Authorization Bearer
> token.
> If not specified, a random token is generated and printed at startup.

*snapshotID*

> Optional.
> Specifies which snapshot to mount.
> If not provided, all snapshots are mounted.

# EXIT STATUS

The **plakar-mount** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Mount all snapshots to a local directory:

	$ plakar mount -to ~/mnt

Mount the latest snapshot to a local directory:

	$ plakar mount -to ~/mnt -latest

Mount a specific snapshot by ID to a directory:

	$ plakar mount -to ~/mnt abc123

Mount all snapshots allowing access by other users:

	$ plakar mount -allow-others -to ~/mnt

Mount snapshots matching a filter (e.g., snapshots with tag "daily-backup"):

	$ plakar mount -to ~/mnt -tag daily-backup

Mount a snapshot to an HTTP endpoint, with a generated token printed at startup:

	$ plakar mount -to http://hostname:8080

Mount a snapshot to an HTTP endpoint with a chosen token, and read a file:

	$ plakar mount -to http://hostname:8080 -token s3cret
	$ curl -H 'Authorization: Bearer s3cret' http://hostname:8080/

Mount a specific snapshot to an HTTP endpoint:

	$ plakar mount -to http://hostname:8080 abc123

# SEE ALSO

plakar(1),
plakar-query(7)

Plakar - October 29, 2026 - PLAKAR-MOUNT(1)
