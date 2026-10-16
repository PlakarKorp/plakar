import React, { PropsWithChildren } from "react";
import {
  Button as RAButton,
  Group,
  Toolbar,
  Separator,
} from "react-aria-components";

import { Typography } from "..";

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
      className="bg-foreground-black m-4 flex w-fit items-center rounded-lg shadow-lg drop-shadow-lg"
    >
      <Group
        aria-label="Items selected"
        className="flex flex-col px-4 py-2 text-center"
      >
        <div className="flex items-baseline gap-1 whitespace-nowrap">
          <Typography className="text-on-brand" weight="medium" size="sm">
            {text}
          </Typography>
          {altText && (
            <Typography className="text-on-brand" size="xs">
              {altText}
            </Typography>
          )}
        </div>
        <Typography
          as={(props) => <RAButton onPress={onUnselect} {...props} />}
          size="xs"
          className="text-tertiary cursor-pointer underline"
        >
          Unselect all
        </Typography>
      </Group>

      {hasChildren && (
        <>
          <Separator
            orientation="vertical"
            className="bg-border-primary w-[1px] shrink-0 self-stretch"
          />

          <Group aria-label="Actions" className="flex gap-2 p-2">
            {children}
          </Group>
        </>
      )}
    </Toolbar>
  );
}
