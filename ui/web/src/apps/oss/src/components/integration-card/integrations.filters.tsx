import { TableFilters } from "@plakar-ui/common/components";
import { Select, TagToggleGroup } from "@plakar-ui/ui/components";
import { useCallback } from "react";
import { Key } from "react-aria-components";

import {
  IntegrationInstallationStatusFilter,
  IntegrationInstallationStatusFilterSchema,
  IntegrationType,
  IntegrationTypeSchema,
} from "~/api/integrations/schema";
import { Route } from "~/routes/_layout/integrations.index";

const TYPE_OPTIONS: Array<{ id: IntegrationType; name: string }> = [
  { id: "storage", name: "Storage" },
  { id: "source", name: "Source" },
  { id: "destination", name: "Destination" },
];

const INSTALLATION_STATUS_OPTIONS: Array<{
  id: IntegrationInstallationStatusFilter;
  name: string;
}> = [
  { id: "installed", name: "Installed" },
  { id: "not-installed", name: "Not installed" },
  { id: "upgradable", name: "To upgrade" },
];

export function IntegrationsFilters() {
  const { search, types, installation_status } = Route.useSearch();
  const navigate = Route.useNavigate();

  const handleSearch = useCallback(
    (search: string) =>
      navigate({
        search: (prev) => ({ ...prev, page: 0, search }),
        replace: true,
      }),
    [navigate]
  );

  const handleTypesChange = useCallback(
    (selected: Set<Key>) => {
      const nextTypes = Array.from(selected)
        .map((k) => IntegrationTypeSchema.safeParse(String(k)))
        .flatMap((r) => (r.success ? [r.data] : []));
      navigate({
        search: (prev) => ({
          ...prev,
          page: 0,
          types: nextTypes.length ? nextTypes : undefined,
        }),
        replace: true,
      });
    },
    [navigate]
  );

  const handleInstallationStatusChange = useCallback(
    (key: string) => {
      const parsed = IntegrationInstallationStatusFilterSchema.safeParse(key);
      navigate({
        search: (prev) => ({
          ...prev,
          page: 0,
          installation_status: parsed.success ? parsed.data : undefined,
        }),
        replace: true,
      });
    },
    [navigate]
  );

  return (
    <TableFilters
      search={search}
      searchPlaceholder="Search integrations..."
      onSearch={handleSearch}
      filters={
        <>
          <Select
            aria-label="Filter by installation"
            placeholder="Filter by installation"
            value={installation_status ?? ""}
            onChange={(key) => handleInstallationStatusChange(String(key))}
            isClearable
            className="min-w-60 max-sm:w-full"
          >
            <Select.ListBox>
              {INSTALLATION_STATUS_OPTIONS.map((option) => (
                <Select.Item
                  key={option.id}
                  id={option.id}
                  name={option.name}
                />
              ))}
            </Select.ListBox>
          </Select>

          <TagToggleGroup
            selectionMode="multiple"
            selectedKeys={types ?? []}
            onSelectionChange={handleTypesChange}
            aria-label="Filter by type"
            className="order-2 max-sm:w-full"
          >
            {TYPE_OPTIONS.map((option) => (
              <TagToggleGroup.Button key={option.id} id={option.id}>
                {option.name}
              </TagToggleGroup.Button>
            ))}
          </TagToggleGroup>
        </>
      }
    />
  );
}
