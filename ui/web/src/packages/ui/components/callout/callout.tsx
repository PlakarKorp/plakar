import { ReactNode } from "react";

import { Icon, IconType, Typography } from "..";

type CalloutProps = {
  icon: IconType;
  title: string;
  description?: string;
  action?: ReactNode;
};

/**
 * Highlighted card announcing one thing and offering one action.
 */
export function Callout({ icon, title, description, action }: CalloutProps) {
  return (
    <div className="border-accent-secondary bg-accent-tertiary flex flex-wrap items-center gap-x-3.5 gap-y-3 rounded-xl border px-4 py-3.5">
      <Icon type={icon} size="md" className="text-accent shrink-0" />
      <div className="min-w-0">
        <Typography weight="medium" className="text-primary">
          {title}
        </Typography>
        {description && (
          <Typography size="sm" className="text-secondary tabular-nums">
            {description}
          </Typography>
        )}
      </div>
      {action && <div className="ml-auto max-sm:w-full">{action}</div>}
    </div>
  );
}
