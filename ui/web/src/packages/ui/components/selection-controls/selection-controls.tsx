import React, { PropsWithChildren } from "react";
import { Group, Toolbar, Separator } from "react-aria-components";

import { IconButton, Typography } from "..";

type SelectionProps = PropsWithChildren<{
  text: string;
  altText?: string;
  onUnselect: () => void;
}>;

export function SelectionControls({
  text,
  altText,
  onUnselect,
  children,
}: SelectionProps) {
  const hasChildren = React.Children.count(children) > 0;

  return (
    <Toolbar
      aria-label="Selection controls"
      className="bg-primary border-primary m-4 flex w-fit items-center gap-2 rounded-xl border p-1.5 shadow-lg"
    >
      <IconButton
        icon="close"
        aria-label="Unselect all"
        tooltip="Unselect all"
        variant="tertiary"
        size="sm"
        onPress={onUnselect}
      />
      <div className="flex items-baseline gap-1.5 pr-1 whitespace-nowrap">
        <Typography weight="medium" size="sm">
          {text}
        </Typography>
        {altText && (
          <Typography size="sm" className="text-tertiary">
            {altText}
          </Typography>
        )}
      </div>

      {hasChildren && (
        <>
          <Separator
            orientation="vertical"
            className="bg-border-secondary h-6 w-px shrink-0"
          />

          <Group aria-label="Actions" className="flex gap-1.5">
            {children}
          </Group>
        </>
      )}
    </Toolbar>
  );
}
