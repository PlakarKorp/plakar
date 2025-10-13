PLAKAR-PKG-MANIFEST.YAML(5) - File Formats Manual

# NAME

**manifest.yaml** - Manifest for plugin assemblation

# DESCRIPTION

The
**manifest.yaml**
file format describes how to package a plugin.
No build or compilation is done, so all executables and other files
must be prepared beforehand.

**manifest.yaml**
must have a top-level YAML object with the following fields:

**name**

> The name of the plugins

**display\_name**

> The displayed name in the UI.

**description**

> A short description of the connectors.

**homepage**

> A link to the homepage.

**license**

> The license of the connectors.

**tag**

> A YAML array of strings for tags that describe the connectors.

**api\_version**

> The API version supported.

**version**

> The plugin version, which doubles as the git tag as well.
> It must follow semantic versioning and have a
> 'v'
> prefix, e.g.
> 'v1.2.3'.

**connectors**

> A YAML array of objects with the following properties:

> **type**

> > The connector type, one of
> > **importer**,
> > **exporter**,
> > or
> > **store**.

> **protocols**

> > An array of YAML strings containing all the protocols that the
> > connector supports.

> **location\_flags**

> > An optional array of YAML strings describing some properties of the
> > connector.
> > These properties are:

> > **localfs**

> > > Whether paths given to this connector have to be made absolute.

> > **file**

> > > Whether this store backend handles a Kloset in a sigle file, for
> > > e.g. a ptar file.

> **executable**

> > Path to the plugin executable.

> **extra\_file**

> > An optional array of YAML string.
> > These are extra files that need to be included in the package.

# EXAMPLES

A sample manifest for the
"fs"
plugin is as follows:

	# manifest.yaml
	name: fs
	display_name: file system connector
	description: file storage but as external plugin
	homepage: https://github.com/PlakarKorp/integration-fs
	license: ISC
	tags: [ fs, filesystem, "local files" ]
	api_version: 1.0.0
	version: 1.0.0
	connectors:
	- type: importer
	  executable: fs-importer
	  protocols: [fs]
	- type: exporter
	  executable: fs-exporter
	  protocols: [fs]
	- type: storage
	  executable: fs-store
	  protocols: [fs]

# SEE ALSO

plakar-pkg-create(1)

Plakar - July 20, 2025 - PLAKAR-PKG-MANIFEST.YAML(5)
