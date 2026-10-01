# Working on plakar

plakar is a backup tool. Data loss is the failure that matters: prefer refusing
to act over acting on a guess.

Go 1.25. The storage engine lives in
[kloset](https://github.com/PlakarKorp/kloset); changes to snapshot or
repository internals usually belong there, not here.

## Build and test

```
make            # build ./plakar
make test       # go test ./...
make coverage   # statement coverage, testing/ helpers excluded
```

Run `gofmt -w` on what you touch and `go vet ./...` before submitting.

## Conventions

Our engineering conventions are a Claude Code plugin, not a document:
[PlakarKorp/claude-standards](https://github.com/PlakarKorp/claude-standards).
Claude Code picks it up from `.claude/settings.json` when you trust this folder
— it injects the conventions and runs gofmt, vet and lint on what you write.

If you use a different agent, read the rules directly in
[`plakar-standards/rules/`](https://github.com/PlakarKorp/claude-standards/tree/main/plakar-standards/rules).

[CONTRIBUTING.md](CONTRIBUTING.md) covers forking, review, and the licensing
constraints on new dependencies.

## What to be careful about

- No `panic` outside `main` and `init`. A backup tool that panics on malformed
  input has lost the user's run.
- Wrap errors with `%w` and context. Handle an error or return it, never both.
- Every goroutine needs an owner that knows when it stops.
- No unbounded reads of input you did not produce.
- Do not edit generated files, `vendor/`, or `go.sum` by hand.
