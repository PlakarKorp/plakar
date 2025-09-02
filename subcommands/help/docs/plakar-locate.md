PLAKAR-LOCATE(1) - General Commands Manual

# NAME

**plakar-locate** - Find filenames in a Plakar snapshot

# SYNOPSIS

**plakar&nbsp;locate**
\[**-name**&nbsp;*name*]
\[**-category**&nbsp;*category*]
\[**-environment**&nbsp;*environment*]
\[**-perimeter**&nbsp;*perimeter*]
\[**-job**&nbsp;*job*]
\[**-tag**&nbsp;*tag*]
\[**-latest**]
\[**-before**&nbsp;*date*]
\[**-since**&nbsp;*date*]
\[**-snapshot**&nbsp;*snapshotID*]
*patterns&nbsp;...*

# DESCRIPTION

The
**plakar locate**
command search all the snapshots to find file names matching any of
the given
*patterns*
and prints the abbreviated snapshot ID and the full path of the
matched files.
Matching works according to the shell globbing rules.

The options are as follows:

**-name** *string*

> Only apply command to snapshots that match
> *name*.

**-category** *string*

> Only apply command to snapshots that match
> *category*.

**-environment** *string*

> Only apply command to snapshots that match
> *environment*.

**-perimeter** *string*

> Only apply command to snapshots that match
> *perimeter*.

**-job** *string*

> Only apply command to snapshots that match
> *job*.

**-tag** *string*

> Only apply command to snapshots that match
> *tag*.

**-latest**

> Only apply command to latest snapshot matching filters.

**-before** *date*

> Only apply command to snapshots matching filters and older than the specified
> date.
> Accepted formats include relative durations
> (e.g. 2d for two days, 1w for one week)
> or specific dates in various formats
> (e.g. 2006-01-02 15:04:05).

**-since** *date*

> Only apply command to snapshots matching filters and created since the specified
> date, included.
> Accepted formats include relative durations
> (e.g. 2d for two days, 1w for one week)
> or specific dates in various formats
> (e.g. 2006-01-02 15:04:05).

**-snapshot** *snapshotID*

> Limit the search to the given snapshot.

# EXAMPLES

Search for files ending in
"wd":

	$ plakar locate '*wd'
	abc123:/etc/master.passwd
	abc123:/etc/passwd

# DIAGNOSTICS

The **plakar-locate** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as invalid parameters, inability to create the
> repository, or configuration issues.

# SEE ALSO

plakar(1),
plakar-backup(1)

# CAVEATS

The patterns may have to be quoted to avoid the shell attempting to
expand them.

Plakar - July 3, 2025 - PLAKAR-LOCATE(1)
