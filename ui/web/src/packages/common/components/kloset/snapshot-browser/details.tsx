import {
  Button,
  DescriptionList,
  Id,
  Spinner,
  Tooltip,
  Typography,
} from "@plakar-ui/ui/components";
import { useInfiniteQuery } from "@tanstack/react-query";
import { ReactNode } from "react";
import {
  Cell,
  Column,
  Row,
  Table,
  TableBody,
  TableHeader,
} from "react-aria-components";

import {
  GetSnapshotPathChunksInfiniteQueryOptions,
  SnapshotType,
  useKlosetAPI,
  VFSEntryType,
} from "@plakar-ui/common/api/kloset";
import { FileMode } from "@plakar-ui/common/components";

type DetailsProps = {
  snapshot: SnapshotType;
  preview: VFSEntryType;
};

export function Details({ preview, snapshot }: DetailsProps) {
  return (
    <div className="flex w-full flex-col gap-8">
      <div className="flex flex-col gap-3">
        <Typography
          as="h3"
          size="sm"
          className="text-primary"
          weight="medium"
          variant="heading"
        >
          General information
        </Typography>
        <GeneralInfo fileEntry={preview} />
      </div>
      {/* Structure only exists for regular files */}
      {preview.file_info.mode.isRegular() && (
        <div className="flex flex-col gap-3">
          <Typography
            as="h3"
            size="sm"
            className="text-primary"
            weight="medium"
            variant="heading"
          >
            Advanced
          </Typography>
          <Structure snapshot={snapshot} fileEntry={preview} />
        </div>
      )}
    </div>
  );
}

type GeneralInfoProps = {
  fileEntry: VFSEntryType;
};

type InfoField = {
  label: string;
  content: ReactNode;
  show?: boolean;
};

function GeneralInfo({ fileEntry }: GeneralInfoProps) {
  const { mode, mod_time, size, uid, username, gid, groupname } =
    fileEntry.file_info;
  const obj = fileEntry.object;

  const fields: InfoField[] = [
    {
      show: mode.isRegular(),
      label: "Checksum",
      content: <Id value={obj?.contentMAC ?? ""} size="sm" weight="medium" />,
    },
    {
      label: "Modified",
      content: (
        <Tooltip label={mod_time.Value}>
          <span role="button">{mod_time.toHumanString()}</span>
        </Tooltip>
      ),
    },
    {
      label: "Size",
      content: (
        <Tooltip label={`${size.Value.toString()} bytes`}>
          <span role="button">{size.Pretty}</span>
        </Tooltip>
      ),
    },
    {
      label: "Mode",
      content: <FileMode mode={mode} size="sm" />,
    },
    {
      show: mode.isSymlink(),
      label: "Symlink target",
      content: fileEntry.symlink_target,
    },
    {
      show: mode.isRegular(),
      label: "Content Type",
      content: obj?.content_type,
    },
    {
      label: "Owner user",
      content: `${uid} ${username}`,
    },
    {
      label: "Owner group",
      content: `${gid} ${groupname}`,
    },
    {
      show: mode.isRegular(),
      label: "Entropy",
      content: obj?.entropy.toFixed(2),
    },
  ];

  return (
    <DescriptionList direction="col" size="sm">
      {fields
        .filter((field) => field.show !== false)
        .map(({ label, content }) => (
          <DescriptionList.Item
            key={label}
            title={label}
            description={content}
          />
        ))}
    </DescriptionList>
  );
}

type StructureProps = {
  snapshot: SnapshotType;
  fileEntry: VFSEntryType;
};

function Structure({ snapshot, fileEntry }: StructureProps) {
  const { api, repository } = useKlosetAPI();
  const query = useInfiniteQuery(
    GetSnapshotPathChunksInfiniteQueryOptions(api, {
      repository,
      snapshot: snapshot.identifier.Value,
      path: fileEntry.path,
    })
  );

  if (query.isError) {
    return null;
  }

  if (query.isPending) {
    return <Spinner />;
  }

  const chunks = query.data.pages.flatMap((page) => page.items);

  const chunksWithOffsets = chunks.map((chunk, idx) => {
    // Size of all the previous chunks
    const offset = chunks
      .slice(0, idx)
      .reduce((sum, curr) => sum + curr.length, 0);

    return {
      ...chunk,
      offset,
    };
  });

  return (
    <div className="flex w-full flex-col items-center gap-4">
      <div className="bg-tertiary w-full shrink-0 overflow-x-auto rounded-sm">
        <Table
          className="w-full table-auto border-separate border-spacing-4 text-left"
          aria-label="Chunks table"
        >
          <TableHeader className="text-tertiary">
            {["Index", "Offset", "Length", "Chunk", "Entropy"].map((header) => (
              <Column
                key={header}
                isRowHeader={header === "Index"}
                className={header === "Index" ? "w-1/9" : "w-2/9"}
              >
                <Typography size="sm">{header}</Typography>
              </Column>
            ))}
          </TableHeader>
          <TableBody className="text-primary">
            {chunksWithOffsets.map((chunk, idx) => (
              <Typography
                key={idx}
                size="sm"
                weight="medium"
                as={(props) => <Row key={idx} {...props} />}
              >
                <Cell>{idx}</Cell>
                <Cell>{chunk.offset}</Cell>
                <Cell>{chunk.length}</Cell>
                <Cell>
                  <Id value={chunk.contentMAC} size="sm" weight="medium" />
                </Cell>
                <Cell>{chunk.entropy.toFixed(2)}</Cell>
              </Typography>
            ))}
          </TableBody>
        </Table>
      </div>

      {query.hasNextPage && (
        <Button
          size="md"
          variant="secondary"
          onPress={() => query.fetchNextPage()}
        >
          Load more
        </Button>
      )}
    </div>
  );
}
