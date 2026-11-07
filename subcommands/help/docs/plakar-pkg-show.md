PLAKAR-PKG-SHOW(1) - General Commands Manual

# NAME

**plakar-pkg-show** - Show installed Plakar plugins

# SYNOPSIS

**plakar&nbsp;pkg&nbsp;show**
\[**-available**]
\[**-devel**]
\[**-long**]

# DESCRIPTION

The
**plakar pkg show**
command shows the currently installed plugins.

The options are as follows:

**-available**

> Instead of installed packages,
> show the set of prebuilt packages available for this system.
> Packages from an additional registry are followed by a tab and the name
> of that registry, see
> plakar-pkg-registry(1).
> Registries that could not be queried, registry entries hidden by a
> package of the same name, and installed packages whose registry was
> removed are reported on the standard error, one per
> line prefixed by
> "warning:".

**-devel**

> Use the integration devel tree.

**-long**

> Show the full package name.

# FILES

*~/.cache/plakar/plugins/*

> Plugin cache directory.
> Respects
> `XDG_CACHE_HOME`
> if set.

*~/.local/share/plakar/plugins*

> Plugin directory.
> Respects
> `XDG_DATA_HOME`
> if set.

# SEE ALSO

plakar-pkg-add(1),
plakar-pkg-build(1),
plakar-pkg-create(1),
plakar-pkg-registry(1),
plakar-pkg-rm(1)

Plakar - September 26, 2026 - PLAKAR-PKG-SHOW(1)
