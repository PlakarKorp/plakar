# Working on the Plakar UI

**pnpm monorepo** managed by Turbo. `apps/oss/` is the web UI served by `plakar ui`.

The app is built and its build is committed in `../build`, so it always ships together with the API it talks to. There is no need to maintain backward compatibility with the API — all API responses must be validated with Zod.

**Typing rules:** Never use `any` or `unknown`. All data must be fully typed in the API layer, and those types must be reused throughout the application.

All commands run from this `ui/web/src/` directory.

## Running the backend

`apps/oss` expects the API on `http://localhost:30112` (`VITE_PLAKAR_URL` in `apps/oss/.env.development`):

```bash
plakar ui -cors -addr 0.0.0.0:30112 -no-spawn -no-auth
```

## Commands

```bash
# Development
pnpm run dev

# Build all packages
pnpm run build

# Run all checks (lint + TypeScript + format + tests) — must always pass
pnpm run check

# Individual checks
pnpm run lint          # ESLint
pnpm run check-ts      # TypeScript compiler check
pnpm run check-format  # Prettier validation
pnpm run format        # Auto-format with Prettier
pnpm run check-knip    # Unused files, exports and dependencies
pnpm run test          # Run tests
```

Scoped to a single package:

```bash
pnpm --filter ./apps/oss test
pnpm --filter ./apps/oss check-ts
```

Tests must always pass. Run `pnpm run check` to validate everything before committing.

TanStack Devtools (Query + Router) are available in dev mode when `VITE_DEVTOOLS=true`.

## Architecture

Shared packages:

- **`packages/ui/`** — Pure presentational component library built on React Aria Components + Tailwind CSS. Has Storybook stories (`apps/plakar-storybook`).
- **`packages/common/`** — Shared business logic: API clients, domain-specific components, and custom hooks.
- **`packages/config/`** — Shared ESLint and TypeScript configurations.

### Common patterns

**Routing:** TanStack Router with file-based routes. Routes live in `src/routes/`; `routeTree.gen.ts` is auto-generated — do not edit manually. Route files use `_layout/$param.lazy.tsx` for code-split routes. Search params are validated with Zod schemas directly in route files.

**Server state:** TanStack Query (React Query v5) with a 5-second `staleTime` default. The `QueryClient` is instantiated in `main.tsx` and provided through context in `__root.tsx`. Retries are disabled for 4xx errors (except 429).

**API layer:** Backend calls go through `packages/common/api/` and `apps/oss/src/api/`. Responses are validated with Zod — no `any` or `unknown` types. API functions are injected via context (e.g., `KlosetAPIContext`) rather than imported directly, so the shared components do not depend on one backend. Types defined in the API layer must be reused everywhere; do not redefine them in components.

**Forms:** TanStack Form via the `useAppForm` hook in `packages/common/hooks/use-app-form.ts`. Zod is used for field-level validation.

**Styling:** Tailwind CSS v4 via the Vite plugin. Use `clsx` for conditional classes. The Prettier plugin sorts Tailwind classes automatically. React Aria's `tailwindcss-react-aria-components` plugin is active for state-based variants (e.g., `rac-focused:`).

**Component layers:**

1. `@plakar-ui/ui` — atomic, unstyled-except-tailwind components; no business logic
2. `@plakar-ui/common/components` — domain components that compose `ui` primitives with API data
3. `apps/oss/src/components/` — page-specific components that use both layers

**Import alias:** `~` maps to `src/` within the app.

**JSDoc:** When writing JSDoc comments, descriptions must start with a capital letter and end with a period. This is enforced by the `jsdoc/match-description` ESLint rule (from `jsdoc.configs["flat/stylistic-typescript"]`). Avoid starting a description with a function or variable name in camelCase — rephrase so it starts with a capital word instead.

**Authentication:** Token stored in `plakar_token` localStorage and sent as an `Authorization: Bearer` header on every request. Login supports magic link email and GitHub OAuth. `APIResponseUnauthorizedError` triggers a redirect to `/login`.
