import { Icon, Typography } from "..";

type ErrorsProps = {
  length?: number;
};

export function Errors({ length }: ErrorsProps) {
  if (!length || length <= 0) return;
  return (
    <Typography size="sm" className="text-error flex items-center gap-1">
      <Icon type="exclamation-circle" size="sm" />
      {length} {length === 1 ? "error" : "errors"}
    </Typography>
  );
}
