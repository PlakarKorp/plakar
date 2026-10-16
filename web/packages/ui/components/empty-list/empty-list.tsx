import { ComponentProps } from "react";

import { Button, Illustration, IllustrationType, Typography } from "..";

type EmptyListProps = {
  title: string;
  description: string;
  actions?: ComponentProps<typeof Button>[];
  illustrationType?: IllustrationType;
  size?: "sm" | "md";
};

export function EmptyList({
  title,
  description,
  actions,
  illustrationType,
  size = "md",
}: EmptyListProps) {
  return (
    <div className="flex flex-col items-center gap-4 p-6">
      {illustrationType && (
        <Illustration
          type={illustrationType}
          className="h-30 w-auto max-w-none shrink-0"
        />
      )}
      <div className="text-primary flex flex-col items-center gap-0.5 text-center">
        <Typography size={size === "sm" ? "md" : "lg"} weight="medium">
          {title}
        </Typography>
        <Typography
          size={size === "sm" ? "sm" : "md"}
          className="text-secondary"
        >
          {description}
        </Typography>
      </div>
      {actions && actions.length > 0 && (
        <div className="flex flex-wrap justify-center gap-2">
          {actions.map((btnProps, index) => (
            <Button key={index} variant="primary" size="md" {...btnProps} />
          ))}
        </div>
      )}
    </div>
  );
}
