import { DescriptionList, Icon, Tooltip } from "@plakar-ui/ui/components";
import { Button, MenuTrigger, Popover } from "react-aria-components";

import { SnapshotType, VFSEntryType } from "@plakar-ui/common/api/kloset";
import { FileMode, Timeline } from "@plakar-ui/common/components";

type DirPreviewProps = {
  snapshot: SnapshotType;
  source: number;
  childEntry: VFSEntryType;
  onSnapshotChange: (snapshotId: string) => void;
};

export function DirectoryPreview({
  snapshot,
  source,
  childEntry,
  onSnapshotChange,
}: DirPreviewProps) {
  const {
    file_info: { mod_time, mode, uid, username, gid, groupname },
  } = childEntry;

  const items = [
    {
      title: "Last modified",
      description: (
        <Tooltip label={mod_time.Value}>
          <span role="button">{mod_time.toHumanString()}</span>
        </Tooltip>
      ),
      tooltip: mod_time.Value,
    },
    {
      title: "Mode",
      description: <FileMode mode={mode} />,
    },
    {
      title: "Owner user",
      description: (
        <>
          {uid} {username}
        </>
      ),
    },
    {
      title: "Owner group",
      description: (
        <>
          {gid} {groupname}
        </>
      ),
    },
    {
      title: "Version",
      description: (
        <MenuTrigger>
          <Button className="flex items-center gap-1 font-mono">
            {snapshot.identifier.toHumanString()}
            <Icon type="caret-down" size="sm" />
          </Button>
          <Popover offset={4}>
            <Timeline
              snapshot={snapshot}
              source={source}
              vfsEntry={childEntry}
              variant="compact"
              onSnapshotChange={onSnapshotChange}
            />
          </Popover>
        </MenuTrigger>
      ),
    },
  ];

  return (
    <DescriptionList direction="row" size="sm">
      {items.map((item) => (
        <DescriptionList.Item key={item.title} {...item} />
      ))}
    </DescriptionList>
  );
}
