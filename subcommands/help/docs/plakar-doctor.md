PLAKAR-DOCTOR(1) - General Commands Manual

# NAME

**plakar-doctor** - Measure the read reliability and latency of a Plakar repository store

# SYNOPSIS

**plakar&nbsp;doctor**
\[**-deep**]
\[**-duration**&nbsp;*time*]
\[**-n**&nbsp;*count*]
\[**-retries**&nbsp;*count*]
\[**-threshold**&nbsp;*percent*]

# DESCRIPTION

The
**plakar doctor**
command reads a random sample of blobs from the repository and reports
how many reads failed, along with a latency distribution and the
throughput.
It helps telling whether a storage connector is healthy before opening
an issue, and can be used from monitoring.

The sample is drawn uniformly at random from the blobs recorded in the
local repository state, whatever their type, so choosing it does not
read anything from the store.
Only the reads of the sampled blobs are measured.
They run in parallel, as many at a time as the
**-concurrency**
option of
plakar(1)
allows.
Every failure is printed with the type of the blob, its mac, the
packfile holding it and the error returned by the store.
When no read fails, the report gives the failure rate the sample rules
out at 95% confidence: a clean run over few blobs cannot tell a healthy
store from one that fails rarely.

The options are as follows:

**-deep**

> Verify the content of each blob against its mac.
> This tells a read that failed from a read that returned the wrong data.
> Blobs of types that are not keyed by the mac of their content are read
> but not verified, and their number is reported.

**-duration** *time*

> Stop reading after
> *time*
> has elapsed.
> Choosing the sample is not counted.
> A value of 0 disables the limit.
> The default is one minute.

**-n** *count*

> Read at most
> *count*
> blobs.
> The default is 60000, enough to catch a failure rate of one read in
> 18000 with 95% probability.
> On a slow store,
> **-duration**
> usually ends the run first.

**-retries** *count*

> Read a failed blob again up to
> *count*
> times, and report whether a retry read it.
> A failure that a retry recovers is transient; one that every retry
> repeats is not.
> The original failure is still counted in the failure rate and the exit
> status.
> The default is 2.

**-threshold** *percent*

> Exit with a non-zero status only if the failure rate is above
> *percent*,
> between 0 and 100.
> The default is 0, any failure is reported as an error.
> The threshold only applies to failed reads: a blob that does not match
> its mac is always an error.

# EXIT STATUS

The
**plakar doctor**
utility exits 0 if the failure rate is not above the threshold,
65 if some blobs did not match their mac,
and 1 if reads failed or another error occurred.

# EXAMPLES

Read blobs for up to a minute and print the results:

	$ plakar doctor

Verify 1000 blobs and tolerate up to one percent of failures:

	$ plakar doctor -n 1000 -deep -threshold 1

# SEE ALSO

plakar(1),
plakar-check(1)

Plakar - September 28, 2026
