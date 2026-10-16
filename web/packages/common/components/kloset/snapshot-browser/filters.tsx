import { Filters, RemovableTags } from "@plakar-ui/ui/components";
import { Key } from "react-aria-components";

import { MIME_FILTERS, TableFilters } from "@plakar-ui/common/components";

type FileFiltersProps = {
  mime?: (typeof MIME_FILTERS)[number];
  filename: string;
  onSearch: (search: string) => void;
  onRemove: (keys: Set<Key>) => void;
  onApply: (filters: Record<string, string[] | null>) => void;
};

export function FileFilters({
  mime,
  filename,
  onSearch,
  onRemove,
  onApply,
}: FileFiltersProps) {
  return (
    <TableFilters
      search={filename}
      searchPlaceholder="Search filename: *.txt, tmp_*..."
      onSearch={onSearch}
      filters={
        <>
          <div className="order-3 w-full md:order-2 md:w-auto">
            {mime && (
              <RemovableTags label="Applied filters" onRemove={onRemove}>
                <RemovableTags.Item id={"mime"} value={mime.name || mime.id} />
              </RemovableTags>
            )}
          </div>
          <div className="order-2 md:order-3">
            <Filters
              items={[
                {
                  id: "mime",
                  label: "File type",
                  options: MIME_FILTERS,
                },
              ]}
              defaultSelected={{ mime: mime ? [mime.id] : null }}
              onApply={onApply}
            />
          </div>
        </>
      }
    />
  );
}
