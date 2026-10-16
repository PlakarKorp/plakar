import { useState } from "react";

import { useKlosetAPI } from "../../../api/kloset";

type UseDownloadSnapshotParams = {
  snapshotId: string;
  paths: string[];
};

export function useDownloadSnapshot({
  snapshotId,
  paths,
}: UseDownloadSnapshotParams) {
  const [isPending, setIsPending] = useState(false);

  const { api, repository } = useKlosetAPI();

  const downloadSnapshot = async () => {
    setIsPending(true);

    const response = await api.downloadSnapshotVFS({
      repository,
      snapshot: snapshotId,
      paths,
    });

    // For the browser to download the file from an AJAX request, we need to
    // create a fake link and simulate a click to it.
    const link = document.createElement("a");
    link.style.display = "none";
    link.target = "_blank";
    link.rel = "noopener noreferrer";
    link.href = response.url;

    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    setIsPending(false);
  };

  return { downloadSnapshot, isPending };
}
