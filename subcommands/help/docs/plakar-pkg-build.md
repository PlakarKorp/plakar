PLAKAR-PKG-BUILD(1) - General Commands Manual

# NAME

**plakar-pkg-build** - Build Plakar plugins from source

# SYNOPSIS

**plakar&nbsp;pkg&nbsp;build&nbsp;*recipe.yaml*&zwnj;**

# DESCRIPTION

The
**plakar pkg build**
fetches the sources and builds the plugin as specified in the given
plakar-pkg-recipe.yaml(5).
If it builds successfully, the resulting plugin will be created in the
current working directory.

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
plakar-pkg-create(1),
plakar-pkg-rm(1),
plakar-pkg-show(1),
plakar-pkg-recipe.yaml(5)

Plakar - July 11, 2025 - PLAKAR-PKG-BUILD(1)
