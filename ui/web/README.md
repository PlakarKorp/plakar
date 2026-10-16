# Plakar UI

The sources of the UI are in [src/](src/). This directory only holds the build, in `build/`, which the `plakar` binary embeds.

To update the build, run the GitHub Action [Update Plakar UI](../../.github/workflows/update-plakar-ui.yml). It builds `src/apps/oss` from a branch of plakar and opens a PR against that branch, updating `ui/web/build`.
