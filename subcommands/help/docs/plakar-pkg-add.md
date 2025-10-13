PLAKAR-PKG-ADD(1) - General Commands Manual

# NAME

**plakar-pkg-add** - Install Plakar plugins

# SYNOPSIS

**plakar&nbsp;pkg&nbsp;add&nbsp;*plugin&nbsp;...*&zwnj;**

# DESCRIPTION

The
**plakar pkg add**
command adds a local or a remote plugin.

If
*plugin*
is an absolute path, or if it starts with
'./',
then it is considered a path to a local plugin file, otherwise
it is downloaded from the Plakar plugin server.
In the latter case, the user must be logged in via the
plakar-login(1)
command.

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

plakar-login(1),
plakar-pkg-build(1),
plakar-pkg-create(1),
plakar-pkg-rm(1),
plakar-pkg-show(1)

Plakar - July 11, 2025 - PLAKAR-PKG-ADD(1)
