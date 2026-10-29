import { FilterPanel } from "@plakar-ui/common/components";
import { SegmentedControl } from "@plakar-ui/ui/components";

import {
  IntegrationInstallationStatusFilter,
  IntegrationInstallationStatusFilterSchema,
  IntegrationType,
  IntegrationTypeSchema,
} from "~/api/integrations/schema";
import {
  type IntegrationCounts,
  shownCount,
} from "~/components/integration-card/use-integration-counts";

const ALL = "all";

const TYPE_OPTIONS: Array<{ id: IntegrationType; label: string }> = [
  { id: "storage", label: "Storage" },
  { id: "source", label: "Source" },
  { id: "destination", label: "Destination" },
];

type IntegrationsFiltersProps = {
  search: string;
  types: IntegrationType[] | undefined;
  status: IntegrationInstallationStatusFilter | undefined;
  counts: IntegrationCounts;
  onSearchChange: (search: string) => void;
  onTypesChange: (types: IntegrationType[] | undefined) => void;
  onStatusChange: (
    status: IntegrationInstallationStatusFilter | undefined
  ) => void;
  onClear: (() => void) | undefined;
};

export function IntegrationsFilters({
  search,
  types,
  status,
  counts,
  onSearchChange,
  onTypesChange,
  onStatusChange,
  onClear,
}: IntegrationsFiltersProps) {
  return (
    <FilterPanel>
      <FilterPanel.Search
        value={search}
        onSearch={onSearchChange}
        placeholder="Search integrations by name…"
        aria-label="Search integrations"
      />
      <FilterPanel.Groups>
        <FilterPanel.Group label="Status">
          <SegmentedControl
            variant="accent"
            ariaLabel="Status"
            selectedId={status ?? ALL}
            onSelect={(id) =>
              onStatusChange(
                IntegrationInstallationStatusFilterSchema.safeParse(id).data
              )
            }
            items={[
              { id: ALL, label: "All", count: shownCount(counts, undefined) },
              {
                id: "installed",
                label: "Installed",
                count: shownCount(counts, "installed"),
                dot: "success",
              },
              {
                id: "upgradable",
                label: "Updates available",
                count: counts.upgradable,
                dot: "warning",
              },
              {
                id: "not-installed",
                label: "Not installed",
                count: counts["not-installed"],
              },
            ]}
          />
        </FilterPanel.Group>
        <FilterPanel.Group label="Type">
          <SegmentedControl
            variant="accent"
            ariaLabel="Type"
            selectedId={types?.length === 1 ? types[0] : ALL}
            onSelect={(id) => {
              const type = IntegrationTypeSchema.safeParse(id);
              onTypesChange(type.success ? [type.data] : undefined);
            }}
            items={[{ id: ALL, label: "All" }, ...TYPE_OPTIONS]}
          />
        </FilterPanel.Group>
      </FilterPanel.Groups>
      <FilterPanel.Footer
        shown={shownCount(counts, status)}
        total={counts.unfiltered}
        noun="integrations"
        onClear={onClear}
      />
    </FilterPanel>
  );
}
