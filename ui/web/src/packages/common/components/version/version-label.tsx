import { Tooltip, Typography } from "@plakar-ui/ui/components";
import clsx from "clsx";

import { type EnrichedVersion } from "../../api/enrich";

type VersionProps = {
  version: EnrichedVersion;
  channel?: boolean;
  onPress?: () => void;
};

export function VersionLabel({
  version,
  channel = false,
  onPress,
}: VersionProps) {
  const shortenedValue = channel ? version.Channel : version.Pretty;

  const text = (
    <Typography
      as={(props) => <button {...props} />}
      onClick={onPress}
      variant="body"
      size="xs"
      className={clsx(
        "text-tertiary",
        onPress && "hover:text-accent cursor-pointer"
      )}
    >
      {shortenedValue}
    </Typography>
  );

  if (onPress) {
    return (
      <Tooltip label={`View changelog · ${version.Value}`}>{text}</Tooltip>
    );
  }

  if (shortenedValue !== version.Value) {
    return <Tooltip label={version.Value}>{text}</Tooltip>;
  }

  return text;
}
