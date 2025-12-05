PLAKAR-QUERY(7) - Miscellaneous Information Manual

# NAME

**plakar-query** - query flags shared among many Plakar subcommands

# DESCRIPTION

What follows is a set of command line arguments that many
plakar(1)
subcommands provide to filter snapshots.

There are two kind of flags:

matchers

> These allow to select snapshots.
> If combined, the result is the union of the various matchers.

filters

> These instead filter the output of the matchers by yielding snapshots
> matching only certain criterias.
> If combined, the result is the intersection of the various filters.

If no matcher is given, all the snapshots are implicitly selected,
and then filtered according to the given filters, if any.

The matchers are divided into:

*	matchers that select snapshots from the last
	*n*
	unit of time:

	**-minutes** *n*

	**-hours** *n*

	**-days** *n*

	**-weeks** *n*

	**-months** *n*

	**-years** *n*

	Or that selects snapshots that were done during the last
	*n*
	days of the week:

	**-mondays** *n*

	**-thuesdays** *n*

	**-wednesdays** *n*

	**-thursdays** *n*

	**-fridays** *n*

	**-saturdays** *n*

	**-sundays** *n*

*	matchers that select at most
	*n*
	snapshots per time period:

	**-per-minute** *n*

	**-per-hour** *n*

	**-per-day** *n*

	**-per-week** *n*

	**-per-month** *n*

	**-per-year** *n*

	**-per-monday** *n*

	**-per-thuesday** *n*

	**-per-wednesday** *n*

	**-per-thursday** *n*

	**-per-friday** *n*

	**-per-saturday** *n*

	**-per-sunday** *n*

The filters are:

**-before** *date*

> Select snapshots older than given
> *date*.
> The date may be in RFC3339 format, as
> "*YYYY*-*mm*-*DD* *HH*:*MM*",
> "*YYYY*-*mm*-*DD* *HH*:*MM*:*SS*",
> "*YYYY*-*mm*-*DD*",
> or
> "*YYYY*/*mm*/*DD*"
> where
> *YYYY*
> is a year,
> *mm*
> a month,
> *DD*
> a day,
> *HH*
> a hour in 24 hour format number,
> *MM*
> minutes and
> *SS*
> the number of seconds.

> Alternatively, human-style intervals like
> "half an hour",
> "a month"
> or
> "2h30m"
> are also accepted.

**-category** *name*

> Select snapshot whose category is
> *name*.

**-environment** *name*

> Select snapshot whose environment is
> *name*.

**-job** *name*

> Select snapshot whose job is
> *name*.

**-latest**

> Select only the latest snapshot.

**-name** *name*

> Select snapshots whose name is
> *name*.

**-perimeter** *name*

> Select snapshots whose perimeter is
> *name*.

**-root** *path*

> Select snapshots whose root directory is
> *path*.
> May be specified multiple time, snapshots are selected if any of the
> given paths matches.

**-since** *date*

> Select snapshots newer than the given
> *date*.
> The accepted format is the same as
> **-before**.

**-tag** *name*

> Select snapshots tagged with
> *name*.
> May be specified multiple times, and multiple tags may be given at the
> same time if comma-separated.
> If a tag name is prefixed with an exclamation mark
> '!',
> the matching is inverted and the snapshot is ignored if it contains
> said tag.

Plakar - November 28, 2025
