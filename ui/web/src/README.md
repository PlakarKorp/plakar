# Plakar UI

The web UI of Plakar, served by `plakar ui`. It is open source, under the ISC license (see [LICENSE](../../../LICENSE)).

The build outputs are static files. They are committed in [ui/web/build](../build), which the `plakar` binary embeds.

## Layout

- `apps/oss`: the Plakar UI
- `apps/plakar-storybook`: Storybook for the shared components
- `packages/ui`: presentational components, built on React Aria Components and Tailwind CSS
- `packages/common`: API clients, domain components and hooks
- `packages/config`: shared ESLint and TypeScript configurations

## Prerequisites

- Node.js 24+
- pnpm 10: `npm install -g pnpm`. It switches itself to the version pinned in `package.json`.

All commands below run from this `ui/web/src/` directory.

## Build

```bash
pnpm install
pnpm run build
```

The build of the Plakar UI is in `apps/oss/dist`. To use it with Plakar, copy it to `../build` and rebuild `plakar`.

## Development

1. Install dependencies:

```bash
pnpm install
```

2. Run the dev server:

```bash
pnpm run dev
```

The server is available at http://localhost:3000. Run the Plakar API on port 30112:

```bash
plakar ui -cors -addr localhost:30112 -no-spawn
```

- `-cors`: allows cross-origin requests, since the UI and API run on different ports.
- `-addr localhost:30112`: binds the API to the port set in [apps/oss/.env.development](apps/oss/.env.development).
- `-no-spawn`: does not open a browser window.
- `-no-auth` (optional): skips the login.

### Commands

- `pnpm run dev`: start the dev server.
- `pnpm run build`: build everything.
- `pnpm run check`: lint, check TypeScript types and formatting, run the tests.
- `pnpm run format`: format the code (configuring auto-format on save is easier).
- `pnpm run check-knip`: look for unused files and dependencies.

## Tech stack

- Vite: build and dev server.
- TypeScript.
- TanStack: router, form, table, query, and pacer for debouncing.
- Zod: validation of the API responses.
- React Aria Components: accessible UI components.
- Turbo: runs the tasks across the workspace.

## Notes

### Auto reload

Most changes reload the page automatically. Changes outside the components (for example to a Zod schema) need a manual reload.

### Run CodeQL locally

To check the build of the Plakar UI for vulnerabilities with CodeQL:

1. Install CodeQL and download the JavaScript analysis pack: `brew install codeql && codeql pack download codeql/javascript-queries`
2. Install the CodeQL extension for VS Code.
3. `cd ./apps/oss`
4. Build: `pnpm run build`. You probably want to add `--minify false` in `package.json` temporarily to make the output readable.
5. `codeql database create ./db --language=javascript`
6. Link the queries pack in the OSS directory: `ln -s ~/.codeql/packages/codeql/javascript-queries .`
7. In VS Code, select the database created at step 5 and run the query you want. If the query does not appear, reload VS Code.

### Create a pnpm patch

To patch a dependency, for example `refractor` because of a CodeQL finding:

```bash
$ pnpm patch refractor
# Edit the files in the directory pnpm prints, then:
$ pnpm patch-commit <directory printed by pnpm patch>
```
