import clsx from "clsx";

import { Icon, IconType, Typography } from "..";
import {
  AvatarShape,
  AvatarSize,
  AvatarStatus,
  AvatarVariant,
} from "./avatar.constants";
import Nestor from "../../assets/nestor-default.svg";

function getInitials(text: string): string {
  const words = text.split(" ");
  if (words.length === 0) return "";
  if (words.length === 1) return words[0].substring(0, 2).toUpperCase();
  return words[0].charAt(0).toUpperCase() + words[1].charAt(0).toUpperCase();
}

type AvatarProps = {
  isLoading?: boolean;
  isDisabled?: boolean;
  status?: AvatarStatus;
  size?: (typeof AvatarSize)[number];
  className?: string;
  variant?: (typeof AvatarVariant)[number];
  shape?: (typeof AvatarShape)[number];
} & (
  | {
      icon?: never;
      text: string;
      url?: string;
    }
  | {
      icon: IconType;
      text?: never;
      url?: never;
    }
);

export function Avatar({
  icon,
  text,
  url,
  isLoading,
  isDisabled,
  status,
  size = "md",
  className,
  variant = "brand",
  shape = "circle",
}: AvatarProps) {
  const contentColor = variant === "group" ? "text-accent" : "text-on-brand";

  return (
    <div
      className={clsx(
        "relative shrink-0",
        { sm: "size-6", md: "size-8 lg:size-9", lg: "size-20" }[size],
        className
      )}
    >
      <div
        className={clsx(
          "relative inline-flex h-full w-full shrink-0 flex-col items-center justify-center overflow-hidden align-top",
          { circle: "rounded-full", square: "rounded-md" }[shape],
          isDisabled
            ? "bg-disable"
            : {
                brand: "bg-accent",
                service: "bg-sky-600",
                group: "bg-accent-tertiary",
              }[variant]
        )}
      >
        {isLoading ? (
          <div className="bg-accent-secondary h-full w-full animate-pulse" />
        ) : icon ? (
          <Icon type={icon} size={ICON_SIZE[size]} className={contentColor} />
        ) : (
          <Typography
            as="div"
            size={TEXT_SIZE[size]}
            weight="medium"
            className={contentColor}
          >
            {url ? (
              <img
                src={url}
                alt={text}
                className="h-full w-full object-cover"
              />
            ) : text ? (
              getInitials(text)
            ) : (
              <img
                alt="Nestor Doe"
                src={Nestor}
                className="absolute top-0 left-1/2 w-[133%] max-w-none -translate-x-[52%]"
              />
            )}
          </Typography>
        )}
      </div>
      {status === "removing" ? (
        <div className="border-background-primary bg-primary absolute -right-0.5 -bottom-0.5 flex size-3.5 items-center justify-center rounded-full border-1">
          <Icon type="circle-minus" size="sm" className="text-error" />
        </div>
      ) : status !== undefined ? (
        <div
          className={clsx(
            "border-background-primary absolute -right-0.5 -bottom-0.5 size-3 rounded-full border-1",
            status === "offline" ? "bg-error" : "bg-success"
          )}
        />
      ) : null}
    </div>
  );
}

const ICON_SIZE = { sm: "md", md: "lg", lg: "6xl" } as const;

const TEXT_SIZE = { sm: "xs", md: "sm", lg: "xl" } as const;
