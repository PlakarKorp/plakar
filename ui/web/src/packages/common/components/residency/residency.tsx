import { IconType, Tag, TagSize, Tooltip } from "@plakar-ui/ui/components";
import { PropsWithChildren, ReactNode } from "react";
import { Button } from "react-aria-components";

import { type EnrichedResidency } from "../../api/enrich";

type ResidencyLabelProps = {
  residency: EnrichedResidency;
  label?: "code" | "name";
};

type ResidencyTagProps = ResidencyLabelProps & {
  size?: (typeof TagSize)[number];
  hasTooltip?: boolean;
  fallback?: string;
  hint?: ReactNode;
};

export function ResidencyLabel({
  residency,
  label = "code",
}: ResidencyLabelProps) {
  if (!residency.alpha2) {
    return null;
  }

  return (
    <span className="inline-flex items-center gap-1">
      <span aria-hidden="true">{residency.flagEmoji}</span>
      {label === "name" ? (
        <span>{residency.name}</span>
      ) : (
        <>
          <span aria-hidden="true">{residency.alpha2}</span>
          <span className="sr-only">{residency.name}</span>
        </>
      )}
    </span>
  );
}

type ResidencyPillProps = PropsWithChildren<{
  size: (typeof TagSize)[number];
  rightIcon?: IconType;
}>;

function ResidencyPill({ size, rightIcon, children }: ResidencyPillProps) {
  return (
    <span className="inline-flex shrink-0">
      <Tag
        variant="default_borderless"
        radius="sqrt"
        size={size}
        rightIcon={rightIcon}
      >
        {children}
      </Tag>
    </span>
  );
}

export function ResidencyTag({
  residency,
  size = "sm",
  label = "code",
  hasTooltip = true,
  fallback,
  hint,
}: ResidencyTagProps) {
  if (!residency.alpha2 && !fallback) {
    return null;
  }

  const tag = (
    <ResidencyPill size={size} rightIcon={hint ? "info-circle" : undefined}>
      {residency.alpha2 ? (
        <ResidencyLabel residency={residency} label={label} />
      ) : (
        fallback
      )}
    </ResidencyPill>
  );

  // Spelling the country out already says what a tooltip would.
  const tooltip =
    hint ?? (hasTooltip && label === "code" ? residency.name : null);

  if (!tooltip) {
    return tag;
  }

  return (
    <Tooltip label={tooltip} maxWidth={280}>
      <Button className="inline-flex cursor-help">{tag}</Button>
    </Tooltip>
  );
}
