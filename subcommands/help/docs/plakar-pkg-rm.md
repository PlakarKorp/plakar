PLAKAR-PKG-RM(1) - General Commands Manual

# NAME

**plakar-pkg-rm** - Uninstall Plakar plugins

# SYNOPSIS

**plakar&nbsp;pkg&nbsp;rm&nbsp;*plugin&nbsp;...*&zwnj;**

# DESCRIPTION

The
**plakar pkg rm**
command removes plugins that have been previously installed with
plakar-pkg-add(1)
command.

The list of plugins can be obtained with
plakar-pkg-show(1).

# EXAMPLES

Removing a plugin:

	$ plakar pkg show
	epic-v1.2.3
	$ plakar pkg rm epic-v1.2.3

# SEE ALSO

plakar-pkg-add(1),
plakar-pkg-build(1),
plakar-pkg-create(1),
plakar-pkg-show(1)

Plakar - July 11, 2025 - PLAKAR-PKG-RM(1)
