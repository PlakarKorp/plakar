PLAKAR-PKG-CREATE(1) - General Commands Manual

# NAME

**plakar-pkg-create** - Package a plugin

# SYNOPSIS

**plakar&nbsp;pkg&nbsp;build&nbsp;*manifest.yaml*&zwnj;**

# DESCRIPTION

The
**plakar pkg create**
assembles a plugin using the provided
plakar-pkg-manifest.yaml(5).

All the files needed for the plugin need to be already available,
i.e. executables must be already be built.

All external files must reside in the same directory as the
*manifest.yaml*
or in subdirectories.

# SEE ALSO

plakar-pkg-add(1),
plakar-pkg-build(1),
plakar-pkg-rm(1),
plakar-pkg-show(1),
plakar-pkg-manifest.yaml(5)

Plakar - July 11, 2025 - PLAKAR-PKG-CREATE(1)
