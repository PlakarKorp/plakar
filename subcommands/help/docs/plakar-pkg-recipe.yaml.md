PLAKAR-PKG-RECIPE.YAML(5) - File Formats Manual

# NAME

**recipe.yaml** - Recipe to build Plakar plugins from source

# DESCRIPTION

The
**recipe.yaml**
file format describes how to fetch and build Plakar plugins.
It must have a top-level YAML object with the following fields:

**name**

> The name of the plugins

**version**

> The plugin version, which doubles as the git tag as well.
> It must follow semantic versioning and have a
> 'v'
> prefix, e.g.
> 'v1.2.3'.

**repository**

> URL to the git repository holding the plugin.

# EXAMPLES

A sample recipe to build the
"fs"
plugin is as follows:

	# recipe.yaml
	name: fs
	version: v1.0.0
	repository: https://github.com/PlakarKorp/integrations-fs

# SEE ALSO

plakar-pkg-build(1)

Plakar - July 11, 2025
