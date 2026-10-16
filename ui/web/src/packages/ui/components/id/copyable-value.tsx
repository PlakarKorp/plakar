import { Typography } from "..";
import { CopyButton } from "./copy-button";

type CopyableValueProps = {
  label: string;
  value: string;
  /** What the copy tooltip names, when the label does not read as a thing to copy. */
  copyLabel?: string;
};

export function CopyableValue({ label, value, copyLabel }: CopyableValueProps) {
  return (
    <div className="flex flex-col gap-1">
      <Typography as="span" size="sm" weight="medium">
        {label}
      </Typography>
      <div className="bg-secondary flex items-start gap-2 rounded-lg p-3">
        <Typography
          as="span"
          size="sm"
          monospace
          className="text-secondary min-w-0 flex-1 break-all"
        >
          {value}
        </Typography>
        <CopyButton
          content={value}
          tooltip={`Copy the ${(copyLabel ?? label).toLowerCase()}`}
        />
      </div>
    </div>
  );
}
