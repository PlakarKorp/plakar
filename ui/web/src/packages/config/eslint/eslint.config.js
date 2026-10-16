import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";
import { defineConfig, globalIgnores } from "eslint/config";
import eslintConfigPrettier from "eslint-config-prettier";
import jsdoc from "eslint-plugin-jsdoc";
import { importX } from "eslint-plugin-import-x";
import eslintPluginUnusedImports from "eslint-plugin-unused-imports";
import storybook from "eslint-plugin-storybook";

export default defineConfig(
  globalIgnores(["dist"]),
  storybook.configs["flat/recommended"],
  jsdoc.configs["flat/contents-typescript"],
  jsdoc.configs["flat/logical-typescript"],
  jsdoc.configs["flat/stylistic-typescript"],
  {
    extends: [
      js.configs.recommended,
      ...tseslint.configs.recommended,
      eslintConfigPrettier,
    ],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
      "import-x": importX,
      "unused-imports": eslintPluginUnusedImports,
    },
    rules: {
      // eslint-plugin-react-hooks v7's `recommended` config now bundles the
      // full React Compiler rule set (16 rules). This repo only wants the
      // two classic hooks rules, so they're picked explicitly instead of
      // spreading the preset.
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "react-refresh/only-export-components": [
        "warn",
        {
          allowConstantExport: true,
          extraHOCs: [
            // recognize Object.assign as a valid HOC.
            // Without it, the following code (from terminal.tsx) triggers a warning:
            //   export const Terminal = Object.assign(TerminalRoot, {
            //     Command,
            //     CommandOutputLine,
            //     Prompt,
            //   });
            // Same fix used by Primer for the same pattern:
            // https://github.com/primer/react/pull/7944
            "assign",
            // TanStack Router route files only export `Route`, e.g.:
            //   export const Route = createFileRoute("/setup")({ component: RouteComponent });
            // createFileRoute/etc. don't return a component, so without this,
            // every local component in the file gets flagged instead.
            // Recommended by the plugin's maintainer:
            // https://github.com/ArnaudBarre/eslint-plugin-react-refresh/issues/102
            "createFileRoute",
            "createLazyFileRoute",
            "createRootRoute",
            "createRootRouteWithContext",
            "withForm",
            "withFieldGroup",
            // JSONForms HOCs
            "withJsonFormsLayoutProps",
            "withJsonFormsControlProps",
            "withJsonFormsEnumProps",
          ],
        },
      ],
      "import-x/order": [
        "error",
        {
          groups: [
            ["builtin", "external"],
            "internal",
            ["parent", "sibling", "index"],
          ],
          "newlines-between": "always", // Adds new lines between groups
          alphabetize: { order: "asc", caseInsensitive: true },
        },
      ],
      "unused-imports/no-unused-imports": "error",
      "unused-imports/no-unused-vars": [
        "warn",
        {
          vars: "all",
          varsIgnorePattern: "^_",
          args: "after-used",
          argsIgnorePattern: "^_",
        },
      ],
      "@typescript-eslint/no-unused-vars": [
        "warn",
        {
          vars: "all",
          varsIgnorePattern: "^_",
          args: "after-used",
          argsIgnorePattern: "^_",
        },
      ],
    },
  }
);
