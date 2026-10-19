import {
  Alert,
  Icon,
  IconButton,
  Spinner,
  Tab,
  Typography,
} from "@plakar-ui/ui/components";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import std_path from "path";
import { useState } from "react";

import { Number } from "@plakar-ui/common/api/enrich";
import {
  GetSnapshotPathQueryOptions,
  RepositoryLocatePathnameQueryOptions,
  SnapshotReaderSignUrlQueryOptions,
  SnapshotType,
  useKlosetAPI,
  VFSEntryType,
} from "@plakar-ui/common/api/kloset";
import {
  Details,
  FileIcon,
  FilePreview,
  FileSummary,
  Timeline,
} from "@plakar-ui/common/components";

type DetailsPanelProps = {
  snapshot: SnapshotType;
  source: number;
  path: string;
  onSnapshotChange: (snapshotId: string) => void;
  onMount?: (mountId: string) => void;
};

export function DetailsPanel({
  snapshot,
  source,
  path,
  onSnapshotChange,
  onMount,
}: DetailsPanelProps) {
  const { api, repository } = useKlosetAPI();
  const query = useQuery(
    GetSnapshotPathQueryOptions(api, {
      repository,
      source,
      snapshot: snapshot.identifier.Value,
      path,
    })
  );

  if (query.isPending) {
    return <Spinner />;
  }

  if (query.isError) {
    return (
      <div className="p-4">
        <Alert variant="error" title="Unable to preview file">
          {query.error.message}
        </Alert>
      </div>
    );
  }

  return (
    <PanelContent
      snapshot={snapshot}
      source={source}
      vfsEntry={query.data.item}
      onSnapshotChange={onSnapshotChange}
      onMount={onMount}
    />
  );
}

type PanelContentProps = {
  snapshot: SnapshotType;
  source: number;
  vfsEntry: VFSEntryType;
  onSnapshotChange: (snapshotId: string) => void;
  onMount?: (mountId: string) => void;
};

function PanelContent({
  snapshot,
  source,
  vfsEntry,
  onSnapshotChange,
  onMount,
}: PanelContentProps) {
  const { api, repository } = useKlosetAPI();
  const { data, isPending, isError } = useQuery(
    SnapshotReaderSignUrlQueryOptions(api, {
      repository,
      snapshot: snapshot.identifier.Value,
      source,
      path: vfsEntry.path,
    })
  );
  const query = useInfiniteQuery(
    RepositoryLocatePathnameQueryOptions(api, snapshot, source, vfsEntry, {
      repository,
      pagination: {
        page: 0,
        perPage: 5,
      },
    })
  );

  const nbVersions = Number(query.data?.pages[0].response.total || 0);

  const items = [
    {
      id: "preview",
      title: "Preview",
      content: (
        <FilePreview
          snapshot={snapshot}
          source={source}
          entry={vfsEntry}
          onMount={onMount}
        />
      ),
    },
    {
      id: "timeline",
      title: "Timeline",
      tag: nbVersions.Value ? nbVersions.Pretty : undefined,
      content: (
        <div className="p-4">
          <Timeline
            snapshot={snapshot}
            source={source}
            vfsEntry={vfsEntry}
            onSnapshotChange={onSnapshotChange}
          />
        </div>
      ),
    },
    {
      id: "details",
      title: "Details",
      content: (
        <div className="p-4">
          <Details snapshot={snapshot} preview={vfsEntry} />
        </div>
      ),
    },
  ] as const;

  const [selected, setSelected] = useState<string>(items[0].id);

  const { mode } = vfsEntry.file_info;
  const isFile = mode.isRegular();
  const isLink = mode.isSymlink();

  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex items-center justify-between px-4">
        <Typography as="h2" className="text-primary flex items-center gap-2">
          {isFile && <FileIcon contentType={vfsEntry.contentType} />}
          {isLink && <Icon type="md-link" size="xl" />}
          {std_path.basename(vfsEntry.path)}
        </Typography>
        <IconButton
          aria-label="Download file"
          tooltip="Download file"
          icon="download"
          variant="tertiary"
          size="sm"
          href={data?.downloadUrl}
          isDisabled={isPending || !data?.downloadUrl || isError}
          download
        />
      </div>
      <div className="hidden md:block">
        <FileSummary vfsEntry={vfsEntry} />
      </div>
      <Tab
        ariaLabel="Preview tabs"
        variant="underlined"
        items={items}
        className="flex h-full min-h-0 flex-col"
        selected={selected}
        onSelectionChange={(id) => setSelected(String(id))}
      />
    </div>
  );
}
