import {
  RepositoryInfoQueryOptions,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import { useLocalStorage } from "@plakar-ui/common/hooks";
import { Accordion, DescriptionList, Hint } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";

import { getDescriptionListItems } from "./utils";

export interface Item {
  label: string;
  value: React.ReactNode;
  tooltip?: string;
}

function DescriptionListItem({ label, value, tooltip }: Item) {
  return (
    <DescriptionList.Item
      title={
        tooltip ? (
          <Hint tooltip={tooltip} color="inherit">
            {label}
          </Hint>
        ) : (
          label
        )
      }
      description={value}
    />
  );
}

type KlosetDescriptionProps = {
  forceClose?: boolean;
};

export function KlosetDescription({ forceClose }: KlosetDescriptionProps) {
  const { api } = useKlosetAPI();
  const { isPending, isError, data } = useQuery(
    RepositoryInfoQueryOptions(api, { repository: undefined })
  );
  const [open, setOpen] = useLocalStorage("kloset-desription-open", true);

  const items = getDescriptionListItems({
    repositoryInfo: data?.item,
    isPending,
  });

  if (isError) return null;

  return (
    <Accordion
      defaultExpandedKeys={open ? ["kloset"] : []}
      expandedKeys={forceClose || !open ? [] : ["kloset"]}
      className="border-secondary overflow-hidden border-t pt-4 pr-2 pl-3"
    >
      <Accordion.Item
        id="kloset"
        title="Kloset store"
        icon="box-archive"
        onExpandedChange={() => setOpen(!open)}
      >
        <div className="pt-1">
          <DescriptionList direction="row" size="xs" singleColumn>
            {items.map((item) => (
              <DescriptionListItem
                key={item.label}
                label={item.label}
                value={item.value}
                tooltip={item.tooltip}
              />
            ))}
          </DescriptionList>
        </div>
      </Accordion.Item>
    </Accordion>
  );
}
