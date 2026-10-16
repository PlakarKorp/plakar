import { Tooltip, Typography } from "@plakar-ui/ui/components";

import { type EnrichedVersion } from "../../api/enrich";

type VersionProps = {
  version: EnrichedVersion;
  channel?: boolean;
};

export function VersionLabel({ version, channel = false }: VersionProps) {
  const shortenedValue = channel ? version.Channel : version.Pretty;

  const text = (
    <Typography
      as={(props) => <button {...props} />}
      variant="body"
      size="xs"
      className="text-tertiary"
    >
      {shortenedValue}
    </Typography>
  );

  if (shortenedValue !== version.Value) {
    return <Tooltip label={version.Value}>{text}</Tooltip>;
  }

  return text;
}
