import clsx from "clsx";
import {
  ComponentPropsWithoutRef,
  createContext,
  PropsWithChildren,
  useContext,
} from "react";

import { Typography } from "..";
import {
  DescriptionListSize,
  DescriptionListDirection,
} from "./description-list.constants";

type DescriptionListProps = ComponentPropsWithoutRef<"dl"> & {
  size?: (typeof DescriptionListSize)[number];
} & (
    | {
        direction?: "col";
        singleColumn?: never;
      }
    | {
        direction: "row";
        singleColumn?: boolean;
      }
  );

const Context = createContext<{
  direction: (typeof DescriptionListDirection)[number];
}>({
  direction: "row",
});

function DescriptionListRoot({
  children,
  direction = "col",
  singleColumn = false,
  size = "md",
  className,
  ...props
}: DescriptionListProps) {
  return (
    <Context.Provider value={{ direction }}>
      <Typography
        as="dl"
        size={size}
        className={clsx(
          "gap-x-4",
          direction === "col" &&
            "grid grid-cols-[auto_1fr] items-baseline gap-y-4",
          direction === "row" &&
            clsx(
              "flex gap-y-2 sm:gap-x-8 md:gap-x-10",
              singleColumn ? "flex-col" : "flex-wrap"
            ),
          className
        )}
        {...props}
      >
        {children}
      </Typography>
    </Context.Provider>
  );
}

type DescriptionListItemProps = {
  title: React.ReactNode;
  description: React.ReactNode;
  truncate?: boolean;
  textOnBrand?: boolean;
};

function DescriptionListItem({
  title,
  description,
  truncate = true,
  textOnBrand = false,
}: DescriptionListItemProps) {
  const { direction } = useContext(Context);
  const isRow = direction === "row";

  const Wrapper = ({ children }: PropsWithChildren) =>
    isRow ? (
      <div className="inline-grid gap-x-4 gap-y-0.5">{children}</div>
    ) : (
      children
    );

  return (
    <Wrapper>
      <dt
        className={clsx(
          textOnBrand ? "text-on-brand-secondary" : "text-tertiary",
          isRow && "row-start-1"
        )}
      >
        {title}
      </dt>
      <dd
        className={clsx(
          textOnBrand ? "text-on-brand" : "text-primary",
          truncate && "truncate",
          isRow ? "row-start-2" : "min-h-6"
        )}
      >
        {description}
      </dd>
    </Wrapper>
  );
}

export const DescriptionList = Object.assign(DescriptionListRoot, {
  Item: DescriptionListItem,
});
