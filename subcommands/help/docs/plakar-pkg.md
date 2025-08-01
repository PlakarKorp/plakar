PLAKAR-PKG(1) - General Commands Manual

# NAME

**plakar-pkg** - List installed Plakar plugins

# SYNOPSIS

**plakar&nbsp;pkg**
\[**-available**]
\[**-long**]

# DESCRIPTION

The
**plakar pkg**
lists the currently installed plugins.

The options are as follows:

**-available**

> Instead of installed packages,
> list the set of prebuilt packages available for this system.

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
plakar-pkg-rm(1)

Plakar - July 11, 2025
