import clsx from "clsx";
import { ListBoxItem } from "react-aria-components";

import { SelectOption, SelectVariant } from "./select";
import { Icon } from "../icon/icon";
import { Typography } from "../typography/typography";

type SelectListBoxItemProps = SelectOption & {
  variant?: Exclude<SelectVariant, "nav" | "heading">;
  isDisabled?: boolean;
  className?: string;
};

export function SelectListBoxItem({
  id,
  name,
  icon,
  item,
  onPress,
  variant = "default",
  isDisabled,
  className,
}: SelectListBoxItemProps) {
  return (
    <ListBoxItem
      id={id}
      value={{ id, name, icon }}
      className={clsx(
        "w-full rounded-sm p-2 wrap-anywhere outline-0",
        {
          default:
            "data-hovered:bg-tertiary data-focused:bg-tertiary data-selected:bg-accent-quaternary data-hovered:data-selected:bg-accent-tertiary data-focused:data-selected:bg-accent-tertiary",
          terminal: "hover:bg-terminal-hover data-selected:bg-terminal-heavy",
        }[variant],
        className
      )}
      onPress={onPress}
      textValue={name}
    >
      {item ?? (
        <div
          className={clsx(
            "flex items-center gap-2",
            {
              default: isDisabled ? "text-tertiary" : "text-primary",
              terminal: "text-terminal-secondary",
            }[variant]
          )}
        >
          {icon && <Icon type={icon} size="md" />}
          <Typography
            className="wrap-anywhere"
            as="span"
            variant="body"
            size="sm"
          >
            {name}
          </Typography>
        </div>
      )}
    </ListBoxItem>
  );
}
