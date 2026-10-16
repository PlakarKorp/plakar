import clsx from "clsx";
import { createContext, ReactNode, useContext } from "react";
import {
  composeRenderProps,
  SelectionIndicator,
  ToggleButton,
  ToggleButtonGroup as AriaToggleButtonGroup,
  ToggleButtonGroupProps as AriaToggleButtonGroupProps,
  ToggleButtonProps,
} from "react-aria-components";

import { Tag } from "../tag/tag";

const AnimationContext = createContext(true);

export type ToggleButtonGroupRootProps = AriaToggleButtonGroupProps & {
  isAnimated?: boolean;
};

export function ToggleButtonGroupRoot({
  className,
  isAnimated = true,
  ...props
}: ToggleButtonGroupRootProps) {
  return (
    <AnimationContext.Provider value={isAnimated}>
      <AriaToggleButtonGroup
        {...props}
        className={clsx(
          "flex w-fit",
          "bg-primary outline-primary rounded-md whitespace-nowrap outline -outline-offset-1",
          className
        )}
      />
    </AnimationContext.Provider>
  );
}

type ToggleButtonGroupItemProps = ToggleButtonProps & {
  tag?: ReactNode;
};

export function ToggleButtonGroupItem({
  tag,
  children,
  ...props
}: ToggleButtonGroupItemProps) {
  const isAnimated = useContext(AnimationContext);

  return (
    <ToggleButton
      {...props}
      className={clsx(
        "group/toggle-button",
        "relative px-2.75 py-2.25",
        "text-secondary cursor-pointer",
        "hover:text-primary",
        "data-selected:text-accent -outline-offset-1",
        "data-disabled:text-disable data-disabled:bg-disable data-disabled:cursor-not-allowed",
        "border-primary not-last:border-r",
        "first:rounded-l-md",
        "last:rounded-r-md",
        "duration-200 motion-safe:transition-colors"
      )}
    >
      {composeRenderProps(children, (children, props) => (
        <>
          <SelectionIndicator
            className={clsx(
              "absolute top-0 -left-[1px] z-1 h-full w-[calc(100%+2px)]",
              "bg-accent-tertiary",
              "border-accent-secondary border",
              "-outline-offset-2 group-data-focus-visible/toggle-button:outline-2",
              "group-first/toggle-button:left-0 group-first/toggle-button:w-[calc(100%+1px)] group-first/toggle-button:rounded-l-md",
              "group-last/toggle-button:-left-[2px] group-last/toggle-button:rounded-r-md",
              "group-only/toggle-button:left-0 group-only/toggle-button:w-full group-only/toggle-button:rounded-md",
              isAnimated
                ? "duration-200 motion-safe:transition-[translate,width,height]"
                : "transition-none"
            )}
          />
          <span className="relative z-2 flex items-center gap-2 text-sm">
            {children}
            {tag !== undefined && !props.isDisabled && (
              <Tag
                size="sm"
                variant={props.isSelected ? "brand" : "default_borderless"}
                radius="sqrt"
              >
                {tag}
              </Tag>
            )}
          </span>
        </>
      ))}
    </ToggleButton>
  );
}

export const ToggleButtonGroup = Object.assign(ToggleButtonGroupRoot, {
  Button: ToggleButtonGroupItem,
});
