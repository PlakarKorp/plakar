import { useMatch, useRouterState } from "@tanstack/react-router";

import { AssistantPage } from "~/api/assistant/schema";

/**
 * Describes the page on screen for the assistant: the snapshot and directory
 * being browsed, the file previewed, the errors listed, or the two snapshots
 * compared. Other pages are named after their first path segment.
 */
export function useAssistantPage(): AssistantPage {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const browse = useMatch({
    from: "/_layout/snapshots/$snapshotId/$source/browse",
    shouldThrow: false,
  });
  const errors = useMatch({
    from: "/_layout/snapshots/$snapshotId/$source/errors",
    shouldThrow: false,
  });
  const compare = useMatch({ from: "/_layout/compare", shouldThrow: false });

  if (browse) {
    return {
      name: "browse",
      snapshot: browse.params.snapshotId,
      path: browse.search.path,
      file: browse.search.preview || undefined,
    };
  }
  if (errors) {
    return {
      name: "errors",
      snapshot: errors.params.snapshotId,
      path: errors.search.path,
    };
  }
  if (compare) {
    return {
      name: "compare",
      older: compare.search.a,
      newer: compare.search.b,
      path: compare.search.path,
    };
  }
  return { name: pathname.split("/")[1] || "snapshots" };
}
