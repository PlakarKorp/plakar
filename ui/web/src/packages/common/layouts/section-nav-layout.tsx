import { NavList, Tab, TabItem } from "@plakar-ui/ui/components";
import { PropsWithChildren } from "react";

type SectionNavLayoutProps = PropsWithChildren<{
  ariaLabel: string;
  items: TabItem[];
  selected?: string;
  onSelect?: (id: string) => void;
}>;

export function SectionNavLayout({
  ariaLabel,
  items,
  selected,
  onSelect,
  children,
}: SectionNavLayoutProps) {
  if (items.filter((item) => !item.isDisabled).length < 2) {
    return children;
  }

  return (
    <div className="flex gap-6 max-md:flex-col max-md:gap-4">
      <NavList aria-label={ariaLabel} className="w-56 shrink-0 max-md:hidden">
        {items.map((item) => (
          <NavList.Item
            key={item.id}
            icon={item.icon}
            label={item.title}
            href={item.href}
            onAction={onSelect && (() => onSelect(item.id))}
            isSelected={item.id === selected}
            isDisabled={item.isDisabled}
          />
        ))}
      </NavList>
      <Tab
        variant="pill"
        ariaLabel={ariaLabel}
        items={items}
        selected={selected}
        onSelectionChange={onSelect && ((key) => onSelect(String(key)))}
        className="md:hidden"
      />
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}
