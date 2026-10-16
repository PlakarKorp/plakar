import { Tooltip, Typography } from "@plakar-ui/ui/components";

import { VFSEntryType } from "@plakar-ui/common/api/kloset";
import { FileMode } from "@plakar-ui/common/components";

type FileSummaryProps = {
  vfsEntry: VFSEntryType;
};

export function FileSummary({ vfsEntry }: FileSummaryProps) {
  const { file_info } = vfsEntry;

  const items = [
    <FileMode mode={file_info.mode} />,
    <Tooltip label={`${file_info.size.Value.toString()} bytes`}>
      <span role="button">{file_info.size.Pretty}</span>
    </Tooltip>,
    <>
      {file_info.uid} {file_info.username}
    </>,
    <>
      {file_info.gid} {file_info.groupname}
    </>,
    <Tooltip label={file_info.mod_time.Value}>
      <span role="button">{file_info.mod_time.toHumanString()}</span>
    </Tooltip>,
  ];

  return (
    <div className="border-accent-heavy ml-4 flex shrink-0 overflow-x-auto border-l-2 pl-2">
      <div className="flex gap-4">
        {items.map((item, idx) => (
          <Typography
            key={idx}
            size="sm"
            weight="medium"
            className="text-primary whitespace-nowrap"
          >
            {item}
          </Typography>
        ))}
      </div>
    </div>
  );
}
