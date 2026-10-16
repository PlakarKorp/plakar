import clsx from "clsx";
import { Key, ToggleButton, ToggleButtonGroup } from "react-aria-components";

import { Icon, IconType, Tooltip, Typography } from "..";

export type SegmentedControlItem = {
  id: string;
  label: string;
  icon?: IconType;
};

export type SegmentedControlProps = {
  items: SegmentedControlItem[];
  selectedId: string;
  onSelect: (id: string) => void;
  ariaLabel: string;
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
        "bg-secondary border-secondary inline-flex flex-row gap-0.5 rounded-lg border p-0.5",
        className
      )}
    >
      {items.map((item) => {
        const segment = (
          <ToggleButton
            key={item.id}
            id={item.id}
            aria-label={hideLabels ? item.label : undefined}
            className={clsx(
              // basis-auto, not the default 0: segments still share any spare
              // width, but a long label sets a floor instead of wrapping.
              "flex flex-1 basis-auto cursor-pointer items-center justify-center gap-1.5 rounded-md px-2 py-1",
              "whitespace-nowrap",
              "text-tertiary data-hovered:text-secondary",
              "data-selected:bg-primary data-selected:text-primary data-selected:shadow-sm",
              "data-focus-visible:outline-focus data-focus-visible:outline-2",
              "data-disabled:text-disable data-disabled:cursor-not-allowed",
              "duration-150 motion-safe:transition-colors"
            )}
          >
            {item.icon && <Icon type={item.icon} size="sm" />}
            {!hideLabels && (
              <Typography as="span" size="sm" weight="medium">
                {item.label}
              </Typography>
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
