# Plakar UI

The sources of the UI are in [src/](src/). The `plakar` binary embeds what `build/` holds when it is built.

`build/` is only committed on release tags. On `main` it is empty, and a binary built from it serves a page explaining how to get the UI ([placeholder.html](placeholder.html)). To build the UI into it, run `make ui` from the root of the repository; it requires Node.js and pnpm.
