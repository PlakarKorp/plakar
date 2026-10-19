import clsx from "clsx";
import { Key, ToggleButton, ToggleButtonGroup } from "react-aria-components";

import { CountBadge, Icon, IconType, StatusDot, Tooltip, Typography } from "..";
import { StatusDotVariant } from "../status-dot/status-dot.constants";

export type SegmentedControlItem = {
  id: string;
  label: string;
  icon?: IconType;
  count?: number;
  dot?: (typeof StatusDotVariant)[number];
};

export type SegmentedControlProps = {
  items: SegmentedControlItem[];
  selectedId: string;
  onSelect: (id: string) => void;
  ariaLabel: string;
  variant?: "default" | "accent";
  hideLabels?: boolean;
  isDisabled?: boolean;
  className?: string;
};

/**
 * Single-choice control showing every option at once, one press away each.
 *
 * With hideLabels, a segment keeps its label as its accessible name and shows
 * it in a tooltip.
 */
export function SegmentedControl({
  items,
  selectedId,
  onSelect,
  ariaLabel,
  variant = "default",
  hideLabels = false,
  isDisabled = false,
  className,
}: SegmentedControlProps) {
  const handleSelectionChange = (keys: Set<Key>) => {
    const [id] = keys;
    if (id !== undefined) {
      onSelect(String(id));
    }
  };

  return (
    <ToggleButtonGroup
      aria-label={ariaLabel}
      selectionMode="single"
      disallowEmptySelection
      selectedKeys={[selectedId]}
      onSelectionChange={handleSelectionChange}
      isDisabled={isDisabled}
      className={clsx(
        "inline-flex flex-row gap-0.5",
        {
          default: "bg-secondary border-secondary rounded-lg border p-0.5",
          accent: "bg-tertiary max-w-full overflow-x-auto rounded-xl p-1",
        }[variant],
        className
      )}
    >
      {items.map((item) => {
        const isDimmed = item.count === 0 && item.id !== selectedId;

        const segment = (
          <ToggleButton
            key={item.id}
            id={item.id}
            aria-label={hideLabels ? item.label : undefined}
            className={clsx(
              "flex cursor-pointer items-center justify-center whitespace-nowrap",
              "data-focus-visible:outline-focus data-focus-visible:outline-2",
              "data-disabled:text-disable data-disabled:cursor-not-allowed",
              "duration-150 motion-safe:transition-colors",
              {
                // basis-auto, not the default 0: segments still share any spare
                // width, but a long label sets a floor instead of wrapping.
                default: clsx(
                  "h-7 flex-1 basis-auto gap-1.5 rounded-md px-2",
                  "text-tertiary data-hovered:text-secondary",
                  "data-selected:bg-primary data-selected:text-primary data-selected:shadow-sm"
                ),
                accent: clsx(
                  "shrink-0 gap-2 rounded-lg px-3 py-1.5",
                  "text-primary data-hovered:bg-primary",
                  "data-selected:bg-accent data-selected:text-on-brand data-selected:data-hovered:bg-accent-hover"
                ),
              }[variant],
              isDimmed && "opacity-50"
            )}
          >
            {({ isSelected }) => (
              <>
                {item.dot && (
                  <StatusDot
                    variant={item.dot}
                    isOnAccent={isSelected && variant === "accent"}
                  />
                )}
                {item.icon && <Icon type={item.icon} size="sm" />}
                {!hideLabels && (
                  <Typography as="span" size="sm" weight="medium">
                    {item.label}
                  </Typography>
                )}
                {item.count !== undefined && (
                  <CountBadge isOnAccent={isSelected && variant === "accent"}>
                    {item.count}
                  </CountBadge>
                )}
              </>
            )}
          </ToggleButton>
        );

        if (!hideLabels) {
          return segment;
        }

        return (
          <Tooltip key={item.id} label={item.label}>
            {segment}
          </Tooltip>
        );
      })}
    </ToggleButtonGroup>
  );
}
