import { SegmentedControl } from "@plakar-ui/ui/components";
import { useRef, useState } from "react";
import { Dialog, Popover } from "react-aria-components";
import { z } from "zod";

import { CreatedRanges, CreatedRangeSearch } from "./created-range";
import { DateBoundPicker } from "./date-bound-picker";

const CreatedRangeSchema = z.enum(CreatedRanges);

const ITEMS = [
  { id: "any", label: "Any time" },
  { id: "24h", label: "24 h" },
  { id: "7d", label: "7 days" },
  { id: "30d", label: "30 days" },
  { id: "custom", label: "Custom", icon: "calendar" as const },
];

type CreatedRangeFilterProps = {
  search: CreatedRangeSearch;
  onChange: (patch: Partial<CreatedRangeSearch>) => void;
};

/**
 * Presets of the creation date, and a custom range picked in a popover under
 * them. The control keeps its width, so the other filters never move.
 */
export function CreatedRangeFilter({
  search,
  onChange,
}: CreatedRangeFilterProps) {
  const triggerRef = useRef<HTMLDivElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const isCustom = search.created === "custom";

  return (
    <div ref={triggerRef} className="flex h-9.5 items-center">
      <SegmentedControl
        items={ITEMS}
        selectedId={search.created ?? "any"}
        onSelect={(id) => {
          const created = CreatedRangeSchema.safeParse(id).data;
          // A second press on "Custom" reopens the range instead of clearing it.
          if (!(isCustom && created === "custom")) {
            onChange({ created, since: undefined, before: undefined });
          }
          setIsOpen(created === "custom");
        }}
        ariaLabel="Created"
      />
      <Popover
        triggerRef={triggerRef}
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        placement="bottom start"
        offset={4}
        className="border-primary bg-primary rounded-lg border shadow-lg"
      >
        <Dialog aria-label="Custom range" className="outline-none">
          <div className="flex w-72 flex-col gap-3 p-4">
            <DateBoundPicker
              label="From"
              value={search.since}
              onChange={(since) => onChange({ since })}
            />
            <DateBoundPicker
              label="To"
              isEnd
              value={search.before}
              onChange={(before) => onChange({ before })}
            />
          </div>
        </Dialog>
      </Popover>
    </div>
  );
}
