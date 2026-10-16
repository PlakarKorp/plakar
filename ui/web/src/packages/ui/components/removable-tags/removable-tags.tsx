import clsx from "clsx";
import { PropsWithChildren } from "react";
import {
  Key,
  Label,
  Tag as AriaTag,
  TagGroup,
  TagGroupProps,
  TagList,
} from "react-aria-components";

import { Icon, IconButton, IconType, Tag, Typography } from "..";

type ListProps = PropsWithChildren<
  TagGroupProps & {
    label: string;
  }
>;

function List({ children, label, ...props }: ListProps) {
  return (
    <TagGroup {...props}>
      <Label slot="label" className="sr-only">
        {label}
      </Label>
      <TagList className="flex flex-wrap gap-1">{children}</TagList>
    </TagGroup>
  );
}

type RemovableTagProps = {
  id: Key;
  value: string;
  icon?: IconType;
  length?: number;
  size?: "sm" | "md";
};

function RemovableTag({
  id,
  value,
  icon,
  length,
  size = "md",
}: RemovableTagProps) {
  return (
    <AriaTag
      id={id}
      className={clsx(
        "border-accent-secondary bg-accent-tertiary inline-flex items-center rounded-sm border px-1",
        {
          sm: "",
          md: "py-1",
        }[size]
      )}
      textValue={value}
    >
      {icon && <Icon type={icon} size="sm" className="text-accent ml-1" />}
      <Typography
        as="span"
        size="sm"
        weight="medium"
        className="text-accent mx-1 my-0.5"
      >
        {value}
      </Typography>
      {length !== undefined && (
        <Tag variant="brand" size="sm" radius="sqrt" className="mr-1 ml-0.5">
          {length}
        </Tag>
      )}
      <IconButton
        slot="remove"
        icon="close"
        variant="accent-tertiary"
        size="sm"
        className="data-disabled:hidden"
      />
    </AriaTag>
  );
}

export const RemovableTags = Object.assign(List, {
  Item: RemovableTag,
});
