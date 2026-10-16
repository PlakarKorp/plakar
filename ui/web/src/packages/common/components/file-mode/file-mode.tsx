import { Tooltip, Typography, TypographySize } from "@plakar-ui/ui/components";
import clsx from "clsx";

import { Mode } from "../../api/enrich";

type FileModeProps = {
  mode: ReturnType<typeof Mode>;
  size?: TypographySize;
};

function getTextClasses(mode: ReturnType<typeof Mode>) {
  if (mode.isSetuid()) return "bg-red-500/80 text-white ring-red-500/80";
  if (mode.isSetgid()) return "bg-orange-500/80 text-white ring-orange-500/80";
  if (mode.isSticky()) return "bg-cyan-500/80 text-white ring-cyan-500/80";
  return "text-primary";
}

export function FileMode({ mode, size = "sm" }: FileModeProps) {
  const hoverText = `${(mode.Value & 0o777).toString(8)}`;
  const hoverExtra = [
    mode.isSetuid() && "SETUID",
    mode.isSetgid() && "SETGID",
    mode.isSticky() && "STICKY",
  ]
    .filter((e) => e)
    .join(" | ");

  return (
    <Tooltip label={`${hoverText} ${hoverExtra}`}>
      <button>
        <Typography
          as="span"
          variant="body"
          size={size}
          weight="medium"
          monospace
          className={clsx("rounded-xs whitespace-nowrap", getTextClasses(mode))}
        >
          {mode.toString()}
        </Typography>
      </button>
    </Tooltip>
  );
}
