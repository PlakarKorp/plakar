import { EmptyList, Typography } from "@plakar-ui/ui/components";
import { ReactNode } from "react";

type IntegrationSectionHeaderProps = {
  title: string;
  total: number | undefined;
  action?: ReactNode;
};

export function IntegrationSectionHeader({
  title,
  total,
  action,
}: IntegrationSectionHeaderProps) {
  return (
    <div className="flex min-h-9.5 flex-wrap items-center justify-between gap-2">
      <Typography as="h2" size="md" weight="semibold" className="text-primary">
        {title}
        {total !== undefined && (
          <span className="text-tertiary font-normal"> · {total}</span>
        )}
      </Typography>
      {action}
    </div>
  );
}

type IntegrationSectionErrorProps = {
  title: string;
  error: Error;
  onRetry: () => void;
};

export function IntegrationSectionError({
  title,
  error,
  onRetry,
}: IntegrationSectionErrorProps) {
  return (
    <EmptyList
      size="sm"
      illustrationType="sad"
      title={`Unable to load the ${title.toLowerCase()} integrations`}
      description={error.message}
      actions={[{ children: "Retry", onPress: onRetry, variant: "secondary" }]}
    />
  );
}
