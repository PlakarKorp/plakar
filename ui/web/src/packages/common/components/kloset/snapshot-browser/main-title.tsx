import { Tag, TagOverflowList } from "@plakar-ui/ui/components";

import { SnapshotType } from "../../../api/kloset";

type MainTitleProps = {
  snapshot: SnapshotType;
  source: number;
};

export function MainTitle({ snapshot, source }: MainTitleProps) {
  return (
    <div className="flex flex-col justify-between gap-3 md:flex-row">
      <span className="flex items-center gap-2">
        <Tag variant="source" radius="sqrt" className="uppercase">
          {snapshot.sources[source]?.importer.type}
        </Tag>
        {snapshot.sources[source]?.importer.origin}
      </span>
      <span className="flex items-center gap-2">
        <Tag variant="transparent" radius="sqrt" leftIcon="folder">
          <span className="font-medium">
            {snapshot.sources[source]?.summary.Total.size.Pretty}
          </span>
        </Tag>

        <TagOverflowList
          items={snapshot.tags}
          variant="blue_borderless"
          radius="sqrt"
          maxRows={4}
        />
      </span>
    </div>
  );
}
