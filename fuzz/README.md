# Fuzz targets

This directory holds the fuzz targets [OSS-Fuzz](https://google.github.io/oss-fuzz/)
runs against plakar: the parsers that read input plakar did not produce
itself. Each is an ordinary Go fuzz test, so they also run as regression
tests under `go test ./...`.

The `f.Add` seeds only count for `go test`: the shim OSS-Fuzz compiles the
targets with makes `F.Add` a no-op, and a target starts there from an empty
corpus unless `seeds.go` builds one for it, which `seeds/main.go` packs into
`<fuzzer>_seed_corpus.zip`. The binary-format targets have one;
`seeds_test.go` checks each seed still decodes.

| Target | What it reads |
|---|---|
| `FuzzConfigFile` | repository configuration files, YAML, JSON or INI |
| `FuzzIgnoreFile` | exclude files in gitignore syntax, plus the path matched |
| `FuzzTimeFlag` | `-since`/`-before` values: dates and human durations |
| `FuzzSingleDash` | the `-flag` to `--flag` rewrite and the pflag parse behind it |
| `FuzzSignifyScope` | trust-store scope lines and the origins matched against them |
| `FuzzSignifyArtifact` | a plugin package with its signify signature and checksum list |
| `FuzzSnapshotHeader` | snapshot headers (msgpack) |
| `FuzzObjectCodec` | object, chunk and cached-path records; MACs in hex and JSON |
| `FuzzVFSEntry` | VFS entries, the per-file records of a snapshot |
| `FuzzPackfileCodec` | packfiles with their index and footer |
| `FuzzStateRecords` | the fixed-size repository state records |
| `FuzzStateConfig` | state configuration entries |
| `FuzzStateMetadata` | state metadata, msgpack and raw header |
| `FuzzSnapshotPath` | `snapshotID:path` arguments |
| `FuzzSortKeys` | `-sort` field lists |

## Running locally

```
go test -fuzz=FuzzPackfileCodec -fuzztime=60s ./fuzz
```

A crash lands in `fuzz/testdata/fuzz/<Target>/` and replays with a plain
`go test ./fuzz` until fixed.

## How OSS-Fuzz builds them

`projects/plakar` in [google/oss-fuzz](https://github.com/google/oss-fuzz)
clones this repository and runs [oss-fuzz-build.sh](oss-fuzz-build.sh),
which compiles every target listed in it with `compile_native_go_fuzzer`
and packs the seed corpora `seeds/main.go` writes. Keeping the target list
here means a new target needs no change on the OSS-Fuzz side. The three
files OSS-Fuzz itself holds are mirrored in [oss-fuzz/](oss-fuzz/).

To build and run them exactly as OSS-Fuzz does, from a checkout of
google/oss-fuzz with Docker running:

```
python3 infra/helper.py build_image plakar
python3 infra/helper.py build_fuzzers plakar /path/to/plakar
python3 infra/helper.py check_build plakar
python3 infra/helper.py run_fuzzer plakar packfile_codec
```

The second command mounts the local checkout over the clone, so uncommitted
targets are what gets built. The build adds its shim to go.mod and go.sum
and puts both back when it exits, and go-118-fuzz-build rewrites each
target file in place while it compiles it and restores it afterwards, so
run `git status` once the build ends and do not interrupt it.
