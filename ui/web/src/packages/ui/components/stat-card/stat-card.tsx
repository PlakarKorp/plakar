import { ReactNode } from "react";

import { Card } from "../card/card";
import { Typography } from "../typography/typography";

type StatCardProps = {
  title: string;
  value: ReactNode;
  label?: string;
  footer?: ReactNode;
};

export function StatCard({ title, value, label, footer }: StatCardProps) {
  return (
    <Card>
      <Card.Header title={title} />
      <Card.Content className="flex flex-col gap-1.5">
        <div className="flex items-baseline gap-2">
          <Typography
            as="p"
            variant="heading"
            weight="medium"
            className="text-primary tabular-nums"
          >
            {value}
          </Typography>
          {label && (
            <Typography size="sm" className="text-tertiary">
              {label}
            </Typography>
          )}
        </div>
        {footer}
      </Card.Content>
    </Card>
  );
}
