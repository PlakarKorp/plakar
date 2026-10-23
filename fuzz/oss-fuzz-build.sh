#!/bin/bash -eu
# Builds plakar's fuzz targets for OSS-Fuzz. projects/plakar/build.sh in
# google/oss-fuzz runs this from the repository root inside
# gcr.io/oss-fuzz-base/base-builder-go, which provides compile_native_go_fuzzer,
# go-118-fuzz-build and the $OUT, $SANITIZER and $LIB_FUZZING_ENGINE it needs.
#
# Add a target by adding its function to fuzz/*_test.go and a line below.
# Function names must not be prefixes of one another: compile_native_go_fuzzer
# locates them with grep.

cd "$(dirname "$0")/.."

# compile_native_go_fuzzer rewrites each fuzz test to import this shim, which
# must be a requirement of the module while it builds. go-118-fuzz-build
# cannot work from an alternate modfile, so go.mod and go.sum are changed in
# place and put back on exit: the builder's clone is disposable, but a
# checkout mounted in for a local run (helper.py build_fuzzers plakar
# /path/to/plakar) is not.
scratch="${WORK:-$(mktemp -d)}"
cp go.mod "$scratch/go.mod.orig"
cp go.sum "$scratch/go.sum.orig"
trap 'cp "$scratch/go.mod.orig" go.mod; cp "$scratch/go.sum.orig" go.sum' EXIT
go get github.com/AdamKorcz/go-118-fuzz-build/testing

package=github.com/PlakarKorp/plakar/fuzz

while read -r function name; do
	compile_native_go_fuzzer "$package" "$function" "$name"
done <<'TARGETS'
FuzzConfigFile config_file
FuzzIgnoreFile ignore_file
FuzzTimeFlag time_flag
FuzzSingleDash single_dash
FuzzSignifyScope signify_scope
FuzzSignifyArtifact signify_artifact
FuzzSnapshotHeader snapshot_header
FuzzObjectCodec object_codec
FuzzVFSEntry vfs_entry
FuzzPackfileCodec packfile_codec
FuzzStateRecords state_records
FuzzStateConfig state_config
FuzzStateMetadata state_metadata
FuzzSnapshotPath snapshot_path
FuzzSortKeys sort_keys
TARGETS

# <name>_seed_corpus.zip for the targets that read a binary format.
go run ./fuzz/seeds "$OUT"
