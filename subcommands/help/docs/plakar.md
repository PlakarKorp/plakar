PLAKAR(1) - General Commands Manual

# NAME

**plakar** - effortless backups

# SYNOPSIS

**plakar**
\[**-config**&nbsp;*path*]
\[**-cpu**&nbsp;*number*]
\[**-keyfile**&nbsp;*path*]
\[**-no-agent**]
\[**-quiet**]
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

**-config** *path*

> Use the configuration at
> *path*.

**-cpu** *number*

> Limit the number of parallel workers
> **plakar**
> uses to
> *number*.
> By default it's the number of online CPUs.

**-keyfile** *path*

> Read the passphrase from the key file at
> *path*
> instead of prompting.
> Overrides the
> `PLAKAR_PASSPHRASE`
> environment variable.

**-no-agent**

> Run without attempting to connect to the agent.

**-quiet**

> Disable all output except for errors.

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

The following commands are available:

**agent**

> Run the plakar agent and configure scheduled tasks, documented in
> plakar-agent(1).

**archive**

> Create an archive from a Kloset snapshot, documented in
> plakar-archive(1).

**backup**

> Create a new Kloset snapshot, documented in
> plakar-backup(1).

**cat**

> Display file contents from a Kloset snapshot, documented in
> plakar-cat(1).

**check**

> Check data integrity in a Kloset store, documented in
> plakar-check(1).

**clone**

> Clone a Kloset store to a new location, documented in
> plakar-clone(1).

**create**

> Create a new Kloset store, documented in
> plakar-create(1).

**destination**

> Manage configurations for the destination connectors, documented in
> plakar-destination(1).

**diff**

> Show differences between files in a Kloset snapshot, documented in
> plakar-diff(1).

**digest**

> Compute digests for files in a Kloset snapshot, documented in
> plakar-digest(1).

**help**

> Show this manpage and the ones for the subcommands.

**info**

> Display detailed information about internal structures, documented in
> plakar-info(1).

**locate**

> Find filenames in a Kloset snapshot, documented in
> plakar-locate(1).

**ls**

> List snapshots and their contents in a Kloset store, documented in
> plakar-ls(1).

**maintenance**

> Remove unused data from a Kloset store, documented in
> plakar-maintenance(1).

**mount**

> Mount Kloset snapshots as a read-only filesystem, documented in
> plakar-mount(1).

**ptar**

> Create a .ptar archive, documented in
> plakar-ptar(1).

**pkg**

> List installed plugins, documented in
> plakar-pkg(1).

**pkg add**

> Install a plugin, documented in
> plakar-pkg-add(1).

**pkg build**

> Build a plugin from source, documented in
> plakar-pkg-build(1).

**pkg create**

> Package a plugin, documented in
> plakar-pkg-create(1).

**pkg rm**

> Unistall a plugin, documented in
> plakar-pkg-rm(1).

**restore**

> Restore files from a Kloset snapshot, documented in
> plakar-restore(1).

**rm**

> Remove snapshots from a Kloset store, documented in
> plakar-rm(1).

**server**

> Start a Plakar server, documented in
> plakar-server(1).

**source**

> Manage configurations for the source connectors, documented in
> plakar-source(1).

**store**

> Manage configurations for storage connectors, documented in
> plakar-store(1).

**sync**

> Synchronize snapshots between Kloset stores, documented in
> plakar-sync(1).

**ui**

> Serve the Plakar web user interface, documented in
> plakar-ui(1).

**version**

> Display the current Plakar version, documented in
> plakar-version(1).

# ENVIRONMENT

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

# FILES

*~/.cache/plakar and* *~/.cache/plakar-agentless*

> Plakar cache directories.

*~/.config/plakar/klosets.yml ~/.config/plakar/sources.yml ~/.config/plakar/destinations.yml*

> Default configuration files.

*~/.plakar*

> Default Kloset store location.

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

Plakar - July 8, 2025
