PLAKAR-TOKEN-CREATE(1) - General Commands Manual

# NAME

**plakar-token-create** - Create a token to authenticate to Plakar services

# SYNOPSIS

**plakar&nbsp;token&nbsp;create**

# DESCRIPTION

The
**plakar token create**
command generates a token that can be used to authenticate with
plakar-login(1).

# EXAMPLES

Generate a token:

	$ plakar token create

and then use it on a different machine to log in automatically:

	$ export PLAKAR_TOKEN=...
	$ plakar login -env

# SEE ALSO

plakar(1),
plakar-login(1),
plakar-service(1)

Plakar - May 5, 2026
