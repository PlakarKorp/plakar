import "~/index.css";
import { KlosetAPI, KlosetAPIContext } from "@plakar-ui/common/api/kloset";
import { Toast } from "@plakar-ui/common/components";
import { LocalStoragePreviewProvider } from "@plakar-ui/common/hooks";
import { ThemeProvider } from "@plakar-ui/common/theme";
import { TanStackDevtools } from "@tanstack/react-devtools";
import { QueryClient } from "@tanstack/react-query";
import { ReactQueryDevtoolsPanel } from "@tanstack/react-query-devtools";
import {
  type NavigateOptions,
  type ToOptions,
  createRootRouteWithContext,
  Outlet,
  useRouter,
} from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { RouterProvider as RARouterProvider } from "react-aria-components";
import { z } from "zod";

import { PlakarAPITokenLocalStorage } from "~/api/plakar/localstorage";
import { FloatingButton } from "~/components/floating-button/floating-button";

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient;
  klosetAPI: KlosetAPI;
}>()({
  validateSearch: z.object({
    plakar_token: z.string().optional(),
  }),
  loaderDeps: ({ search: { plakar_token } }) => {
    return { plakar_token };
  },

  loader: async ({ deps }) => {
    if (deps.plakar_token) {
      PlakarAPITokenLocalStorage.store(deps.plakar_token);
    }
  },

  component: Component,
});

// See: https://react-spectrum.adobe.com/react-aria/routing.html#tanstack-router
declare module "react-aria-components" {
  interface RouterConfig {
    href: ToOptions;
    routerOptions: Omit<NavigateOptions, keyof ToOptions>;
  }
}

function Component() {
  const router = useRouter();
  const { klosetAPI } = Route.useRouteContext();

  return (
    <RARouterProvider
      navigate={(href, opts) => router.navigate({ ...href, ...opts })}
      useHref={(href) =>
        typeof href === "string" ? href : router.buildLocation(href).href
      }
    >
      <ThemeProvider>
        <LocalStoragePreviewProvider>
          <KlosetAPIContext.Provider
            value={{ api: klosetAPI, repository: undefined }}
          >
            <Toast />
            <Outlet />
            <FloatingButton />
          </KlosetAPIContext.Provider>
        </LocalStoragePreviewProvider>
      </ThemeProvider>

      {!!import.meta.env.VITE_DEVTOOLS && (
        <TanStackDevtools
          plugins={[
            {
              name: "TanStack Query",
              render: <ReactQueryDevtoolsPanel />,
            },
            {
              name: "TanStack Router",
              render: <TanStackRouterDevtoolsPanel />,
            },
          ]}
        />
      )}
    </RARouterProvider>
  );
}
