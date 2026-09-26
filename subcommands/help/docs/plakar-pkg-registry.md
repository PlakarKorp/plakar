PLAKAR-PKG-REGISTRY(1) - General Commands Manual

# NAME

**plakar-pkg-registry** - Manage additional Plakar plugin registries

# SYNOPSIS

**plakar&nbsp;pkg&nbsp;registry**
**add**
*name*
*url*  
**plakar&nbsp;pkg&nbsp;registry**
**rm**
*name*  
**plakar&nbsp;pkg&nbsp;registry**
**list**

# DESCRIPTION

The
**plakar pkg registry**
command manages the plugin registries consulted in addition to the
official Plakar plugin server.
A registry is laid out like the official distribution tree: the
integrations index at its root and one directory per edition below it.

plakar-pkg-add(1)
looks a plugin up in the official catalog first, then in each registry.
The official catalog always wins on a name clash: a registry entry named
like an official plugin is ignored, and
**plakar pkg show** **-available**
reports it with a warning.
That command also shows the registry each available plugin comes from.
Registries are consulted in the order of their names.
A plugin installed from a registry is later updated from that same
registry.

The subcommands are as follows:

**add** *name* *url*

> Add the registry
> *name*
> served at
> *url*.
> *name*
> may only contain letters, digits,
> '-',
> and
> '\_',
> and must not be
> **official**.
> *url*
> must use https and have no credentials, query or fragment.

> Packages from a registry are only installed if they are signed by a key
> trusted for it.
> If no trusted key covers
> *url*,
> a reminder explains how to trust one: copy the registry's public key to
> *~/.config/plakar/trust/*&zwnj;*key*&zwnj;*.pub*
> and write
> *url*
> on a line of
> *~/.config/plakar/trust/*&zwnj;*key*&zwnj;*.scope*.

**rm** *name*

> Remove the registry
> *name*.
> The plugins installed from it are left installed, but can no longer be
> updated until a registry with the same URL is added again.

**list**

> List the official plugin server, then each registry, one per line, as
> its name and URL separated by a tab.

# FILES

*~/.config/plakar/registries.yml*

> The configured registries.
> Only the
> **add**
> subcommand enforces https: a hand-edited file may contain http URLs.
> Respects
> `XDG_CONFIG_HOME`
> if set.

*~/.config/plakar/trust/*

> Trusted keys and their scopes.

# EXAMPLES

Adding a registry and listing the configured ones:

	$ plakar pkg registry add lab https://plakar.example.org/
	$ plakar pkg registry list
	official	https://plakar.io/dist/plugins/kloset/
	lab	https://plakar.example.org/

The corresponding
*registries.yml*:

	version: v1.0.0
	registries:
	  lab:
	    url: https://plakar.example.org/

# SEE ALSO

plakar-pkg-add(1),
plakar-pkg-rm(1),
plakar-pkg-show(1)

Plakar - September 26, 2026 - PLAKAR-PKG-REGISTRY(1)
