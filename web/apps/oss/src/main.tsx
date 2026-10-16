import {
  APIResponseError,
  APIResponseNotImplementedError,
  APIResponseUnauthorizedError,
} from "@plakar-ui/common/api/errors";
import { NotFoundError } from "@plakar-ui/common/components";
import { Spinner } from "@plakar-ui/ui/components";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { ZodError } from "zod";

import {
  downloadSnapshotVFS,
  generateSnapshotSignedURL,
  getKlosetURL,
  getRepositoryImporterTypes,
  getRepositoryInfo,
  getRepositoryLocatePathname,
  getSnapshot,
  getSnapshotPath,
  getSnapshotPathChildren,
  getSnapshotPathChunks,
  getSnapshotPathErrors,
  getSnapshotPathSearch,
  getSnapshotsList,
} from "./api/kloset/api";
import UnexpectedError from "./components/error/internal-error";
import { routeTree } from "./routeTree.gen";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Make useQuery throw these errors instead of returning them in the result.
      //
      // Note that useSuspenseQuery always throws errors. This is only for
      // useQuery/useInfiniteQuery:
      // - when false, errors are returned in the result
      // - when true, they are thrown and can be caught by an error boundary.
      throwOnError: (error) => {
        return error instanceof APIResponseUnauthorizedError;
      },
      // Disable retry for 4xx errors (client errors) except 429 (rate limit).
      retry: (count, error) => {
        if (error instanceof TypeError) {
          return false;
        }
        if (error instanceof ZodError) {
          return false;
        }
        if (error instanceof APIResponseError) {
          if (error.code === 429) {
            return true;
          }
          if (error.code >= 400 && error.code < 500) {
            return false;
          }
          if (error instanceof APIResponseNotImplementedError) {
            return false;
          }
        }
        return count < 3;
      },

      // Consider data fresh for 5 seconds to avoid refetching on every mount within that time.
      // This is useful to allow children components to query the data already
      // fetched by the parent route without triggering additional requests.
      staleTime: 5 * 1000,
    },
  },
});

const router = createRouter({
  routeTree,
  context: {
    queryClient,
    klosetAPI: {
      getKlosetURL: getKlosetURL,
      getRepositoryInfo: getRepositoryInfo,
      getRepositoryImporterTypes: getRepositoryImporterTypes,
      getRepositoryLocatePathname: getRepositoryLocatePathname,
      getSnapshotsList: getSnapshotsList,
      getSnapshot: getSnapshot,
      getSnapshotPath: getSnapshotPath,
      getSnapshotPathChildren: getSnapshotPathChildren,
      getSnapshotPathSearch: getSnapshotPathSearch,
      getSnapshotPathErrors: getSnapshotPathErrors,
      downloadSnapshotVFS: downloadSnapshotVFS,
      getSnapshotPathChunks: getSnapshotPathChunks,
      generateSnapshotSignedURL: generateSnapshotSignedURL,
    },
  },
  defaultPreload: "intent",
  defaultPendingMs: 300,
  defaultPendingComponent: Spinner,
  defaultErrorComponent: UnexpectedError,
  defaultNotFoundComponent: () => <NotFoundError href={{ to: "/" }} />,
  // Since we're using React Query, we don't want loader calls to ever be stale
  // This will ensure that the loader is always called when the route is preloaded or visited
  defaultPreloadStaleTime: 0,
});

// Register the router instance for type safety
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>
);
