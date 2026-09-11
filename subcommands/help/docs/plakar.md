PLAKAR(1) - General Commands Manual

# NAME

**plakar** - effortless backups

# SYNOPSIS

**plakar**
\[**-concurrency**&nbsp;*number*]
\[**-configdir**&nbsp;*dir*]
\[**-cachedir**&nbsp;*dir*]
\[**-datadir**&nbsp;*dir*]
\[**-cpu**&nbsp;*number*]
\[**-json**]
\[**-keyfile**&nbsp;*path*]
\[**-quiet**]
\[**-silent**]
\[**-stdio**]
\[**-time**]
\[**-trace**&nbsp;*subsystems*]
\[**at**&nbsp;*kloset*]
*subcommand&nbsp;...*

# DESCRIPTION

**plakar**
is a tool to create distributed, versioned backups with compression,
encryption, and data deduplication.

By default,
**plakar**
operates on the Kloset store at
*~/.plakar*.
This can be changed either by using the
**at**
option.

The following options are available:

**-concurrency** *number*

> Set the maximum number of parallel tasks for faster processing.
> Defaults to the CPU count.

**-configdir** *dir*

> Specify an alternate configuration directory.
> Defaults to
> *~/.config/plakar*.

**-cachedir** *dir*

> Specify an alternate cache directory.
> Defaults to
> *~/.cache/plakar*.

**-datadir** *dir*

> Specify an alternate data directory.
> Defaults to
> *~/.local/share/plakar*.

**-cpu** *number*

> Limit the number of parallel workers
> **plakar**
> uses to
> *number*.
> By default it's the number of online CPUs.

**-json**

> Use newline-delimited JSON as output format for some subcommands.

**-keyfile** *path*

> Read the passphrase from the key file at
> *path*
> instead of prompting.
> Overrides the
> `PLAKAR_PASSPHRASE`
> environment variable.

**-quiet**

> Disable all output except for errors.

**-silent**

> Disable all output.

**-stdio**

> Use text lines as output format for some subcommands instead of the
> default ncurses frontend.
> Enabled by default when the standard output is not a terminal.

**-time**

> Report the time the subcommand took to run.

**-trace** *subsystems*

> Display trace logs.
> *subsystems*
> is a comma-separated series of keywords to enable the trace logs for
> different subsystems:
> **all**, **trace**, **repository**, **snapshot** and **server**.

**at** *kloset*

> Operates on the given
> *kloset*
> store.
> It could be a path, an URI, or a label in the form
> "@*name*"
> to reference a configuration created with
> plakar-store(1).

## General Commands

**help**

> Show this manpage and the ones for the subcommands.

**login**

> Authenticate to Plakar services, refer to
> plakar-login(1).

**logout**

> Log out from Plakar services, refer to
> plakar-logout(1).

**service**

> Manage additional Plakar services that require you to be logged in, refer to
> plakar-service(1).

**token create**

> Generate a token to interact with Plakar services, refer to
> plakar-token-create(1).

**version**

> Display the current Plakar version, refer to
> plakar-version(1).

## Configuration management

**destination**

> Manage configurations for the destination connectors, refer to
> plakar-destination(1).

**source**

> Manage configurations for the source connectors, refer to
> plakar-source(1).

**store**

> Manage configurations for storage connectors, refer to
> plakar-store(1).

## Kloset management

**check**

> Check data integrity in a Kloset store, refer to
> plakar-check(1).

**create**

> Create a new Kloset store, refer to
> plakar-create(1).

**info**

> Display detailed information about internal structures, refer to
> plakar-info(1).

**maintenance**

> Remove unused data from a Kloset store, refer to
> plakar-maintenance(1).

**prune**

> Prune snapshots according to a policy, refer to
> plakar-prune(1).

**ptar**

> Create a .ptar archive, refer to
> plakar-ptar(1).

**server**

> Start a Plakar server, refer to
> plakar-server(1).

**sync**

> Synchronize snapshots between Kloset stores, refer to
> plakar-sync(1).

**ui**

> Serve the Plakar web user interface, refer to
> plakar-ui(1).

## Snapshot management

**archive**

> Create an archive from a Kloset snapshot, refer to
> plakar-archive(1).

**backup**

> Create a new Kloset snapshot, refer to
> plakar-backup(1).

**cat**

> Display file contents from a Kloset snapshot, refer to
> plakar-cat(1).

**diff**

> Show differences between files in a Kloset snapshot, refer to
> plakar-diff(1).

**digest**

> Compute digests for files in a Kloset snapshot, refer to
> plakar-digest(1).

**dup**

> Duplicate an existing snapshot with a different ID, refer to
> plakar-dup(1).

**locate**

> Find filenames in a Kloset snapshot, refer to
> plakar-locate(1).

**ls**

> List snapshots and their contents in a Kloset store, refer to
> plakar-ls(1).

**mount**

> Mount Kloset snapshots as a read-only filesystem, refer to
> plakar-mount(1).

**restore**

> Restore files from a Kloset snapshot, refer to
> plakar-restore(1).

**rm**

> Remove snapshots from a Kloset store, refer to
> plakar-rm(1).

## Plugin handling

**pkg add**

> Install a plugin, refer to
> plakar-pkg-add(1).

**pkg build**

> Build a plugin from source, refer to
> plakar-pkg-build(1).

**pkg create**

> Package a plugin, refer to
> plakar-pkg-create(1).

**pkg rm**

> Uninstall a plugin, refer to
> plakar-pkg-rm(1).

**pkg show**

> List installed plugins, refer to
> plakar-pkg-show(1).

# ENVIRONMENT

`PLAKAR_INSECURE_PLAINTEXT`

> Must be set to work with a Kloset store that is not encrypted.
> Without it, every command that would read from or write to an unencrypted
> store refuses to run.

`PLAKAR_PASSPHRASE`

> Passphrase to unlock the Kloset store; overrides the one from the configuration.
> If set,
> **plakar**
> won't prompt to unlock.
> The option
> **keyfile**
> overrides this environment variable.

`PLAKAR_REPOSITORY`

> Reference to the Kloset store.

`PLAKAR_TOKEN`

> Token to authenticate for Plakar services.

# FILES

*~/.cache/plakar*

> Plakar cache directories.

*~/.config/plakar/destinations.yml*

> Restore destinations configuration.

*~/.config/plakar/sources.yml*

> Backup sources configuration.

*~/.config/plakar/stores.yml*

> Kloset stores configuration.

*~/.plakar*

> Default Kloset store location.

# EXIT STATUS

The following exit codes are aligned with
sysexits(3)
where applicable:

0

> Command completed successfully.

1

> A general error occurred.

64 (EX\_USAGE)

> Invalid command-line arguments or flags.

65 (EX\_DATAERR)

> Data integrity check failed (corrupted chunks, verification mismatch).

66 (EX\_NOINPUT)

> The repository could not be opened or located.

77 (EX\_NOPERM)

> Authentication or decryption failure (wrong passphrase, missing keyfile), or
> an unencrypted store used without
> `PLAKAR_INSECURE_PLAINTEXT`.

78 (EX\_CONFIG)

> Incompatible repository version.

# EXAMPLES

Create an encrypted Kloset store at the default location:

	$ plakar create

Create an encrypted Kloset store on AWS S3:

	$ plakar store add mys3bucket \
	    location=s3://s3.eu-west-3.amazonaws.com/backups \
	    access_key="access_key" \
	    secret_access_key="secret_key"
	$ plakar at @mys3bucket create

Create a snapshot of the current directory on the @mys3bucket Kloset store:

	$ plakar at @mys3bucket backup

List the snapshots of the default Kloset store:

	$ plakar ls

Restore the file
"notes.md"
in the current directory from the snapshot with id
"abcd":

	$ plakar restore -to . abcd:notes.md

Remove snapshots older than 30 days:

	$ plakar rm -before 30d

Plakar - May 5, 2026 - PLAKAR(1)
