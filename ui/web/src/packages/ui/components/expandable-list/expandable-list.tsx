import { Button, Disclosure, DisclosurePanel } from "react-aria-components";

import { Icon, IconType, Tag, Typography } from "..";

type Item = {
  value: string;
  subValue?: string;
} & (
  | {
      icon?: IconType;
      tag?: never;
    }
  | {
      icon?: never;
      tag?: string;
    }
);

type ExpandableListProps = {
  items: Item[];
};

export function ExpandableList({ items }: ExpandableListProps) {
  const firstItem = items[0];
  const collapsedItems = items.slice(1);

  if (items.length <= 1) {
    return <ListItem {...firstItem} />;
  }

  return (
    <Disclosure className="group">
      <Button
        slot="trigger"
        className="flex max-w-full min-w-0 items-start gap-2"
      >
        <ListItem {...firstItem} />
        <div className="bg-foreground-black text-on-brand flex shrink-0 items-center rounded-full px-1.5 py-0.5">
          <Typography
            variant="body"
            size="xs"
            className="px-1 group-[&[data-expanded=true]]:hidden"
          >
            +{items.length - 1}
          </Typography>
          <div className="flex h-5 items-center px-1">
            <div className="transition-all group-[&[data-expanded=true]]:rotate-180">
              <Icon type="caret-down" size="sm" />
            </div>
          </div>
        </div>
      </Button>
      <DisclosurePanel>
        <div className="mt-4 flex flex-col gap-4">
          {collapsedItems.map((item, index) => (
            <ListItem key={`${item.value}${index}`} {...item} />
          ))}
        </div>
      </DisclosurePanel>
    </Disclosure>
  );
}

type ListItemProps = Item;

function ListItem({ icon, tag, value, subValue }: ListItemProps) {
  return (
    <div className="flex min-w-0 flex-col gap-1">
      {value && (
        <div className="flex items-center gap-2">
          {icon && <Icon type={icon} size="lg" />}
          {tag && (
            <Tag variant="source" radius="sqrt" className="max-w-1/3" size="sm">
              {tag.toUpperCase()}
            </Tag>
          )}
          <Typography
            variant="body"
            size="md"
            className="text-primary truncate"
          >
            {value}
          </Typography>
        </div>
      )}
      {subValue && (
        <Typography
          variant="body"
          size="sm"
          className="text-secondary truncate text-left"
        >
          {subValue}
        </Typography>
      )}
    </div>
  );
}
