import clsx from "clsx";
import { ReactNode } from "react";
import {
  Link as AriaLink,
  LinkProps as AriaLinkProps,
  Button as AriaButton,
  ButtonProps as AriaButtonProps,
} from "react-aria-components";

type SharedProps = "children" | "className";

type AnchorProps = Omit<AriaLinkProps, SharedProps> & { onPress?: never };
type ButtonProps = Omit<AriaButtonProps, SharedProps> & {
  href?: never;
  target?: never;
};

type LinkProps = (AnchorProps | ButtonProps) & {
  children?: ReactNode;
  className?: string;
  inheritColor?: boolean;
};

function isAnchor(props: AnchorProps | ButtonProps): props is AnchorProps {
  return props.href !== undefined;
}

export function Link({
  children,
  className,
  inheritColor,
  ...props
}: LinkProps) {
  if (props.href === undefined && props.onPress === undefined) {
    return <>{children}</>;
  }

  const classes = clsx(
    inheritColor ? "text-inherit" : "text-accent",
    "hover:text-accent-hover cursor-pointer hover:underline",
    className
  );

  if (isAnchor(props)) {
    return (
      <AriaLink className={classes} {...props}>
        {children}
      </AriaLink>
    );
  }

  return (
    <AriaButton className={classes} {...props}>
      {children}
    </AriaButton>
  );
}
