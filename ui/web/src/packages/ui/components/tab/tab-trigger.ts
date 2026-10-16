import clsx from "clsx";

export type TabVariant = "underlined" | "pill";

/**
 * Classes shared by the tabs of `Tab` and the trigger of `Tab.Dropdown`, so
 * a dropdown sitting in a tab row is indistinguishable from a real tab.
 *
 * Selection is resolved here rather than through `data-selected:` variants,
 * because the dropdown trigger is a button and never carries that attribute.
 * Conflicting utilities must therefore never be emitted together: which one
 * wins would come down to stylesheet order, not to the order below.
 */
export function getTabTriggerClassName({
  variant,
  isSelected,
  isDisabled,
}: {
  variant: TabVariant;
  isSelected: boolean;
  isDisabled?: boolean;
}) {
  return clsx(
    "relative flex items-center justify-center gap-2 whitespace-nowrap -outline-offset-1",

    variant === "underlined" && {
      "border-b-2 border-transparent px-4 py-3": true,
      "text-secondary hover:border-secondary hover:text-primary":
        !isSelected && !isDisabled,
      "text-primary": isSelected,
    },

    variant === "pill" && {
      "h-9 rounded-lg px-2 py-1": true,
      "bg-primary text-primary": isSelected,
      "hover:bg-tertiary-hover": !isSelected && !isDisabled,
      "text-tertiary": !isSelected && !isDisabled,
    },

    isDisabled ? "cursor-not-allowed" : "cursor-pointer",
    isDisabled && !isSelected && "text-disable"
  );
}
