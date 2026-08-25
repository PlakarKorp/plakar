PLAKAR-PKG-ADD(1) - General Commands Manual

# NAME

**plakar-pkg-add** - Install Plakar plugins

# SYNOPSIS

**plakar&nbsp;pkg&nbsp;add**
\[**-allow-unsigned**]
\[**-u**]
*plugin&nbsp;...*

# DESCRIPTION

The
**plakar pkg add**
command adds a local or a remote plugin.

If
*plugin*
matches an existing local file, it is installed directly.
Otherwise, it is treated as a recipe name and downloaded from the Plakar plugin
server which requires a login via the
plakar-login(1)
command.

Installing plugins without logging in is possible via the
plakar-pkg-build(1)
command, provided you have the necessary dependencies to build
it locally
(currently, official plugins require make and a working Go toolchain).

To install a specific version of a plugin, use the
*name*@*version*
syntax.

The options are as follows:

**-allow-unsigned**

> Install packages that carry no signature.
> By default, package are expected to be signed.

**-u**

> Update the specific plugins.
> If none are given, attempt to update all the installed ones.

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

Plakar - August 25, 2026 - PLAKAR-PKG-ADD(1)
