import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Tag, Typography } from "..";
import { CommandPalette } from "./command-palette";

const ITEMS = [
  { id: "stores", label: "Stores", icon: "box-archive", section: "Pages" },
  { id: "sources", label: "Inventories", icon: "cloud", section: "Pages" },
  { id: "backup", label: "Start a backup", icon: "upload", section: "Actions" },
  {
    id: "s3-prod",
    label: "s3-prod-eu",
    icon: "box-archive",
    section: "Stores",
    description: "s3://backups-eu-west-3",
  },
] as const;

function Template({ isLoading = false }: { isLoading?: boolean }) {
  const [isOpen, setIsOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [scope, setScope] = useState<string>("all");
  const [focused, setFocused] = useState<string>();

  const visible = ITEMS.filter((item) =>
    item.label.toLowerCase().includes(query.toLowerCase())
  );
  const sections = [...new Set(visible.map((item) => item.section))];

  return (
    <div className="w-60">
      <CommandPalette.Trigger onPress={() => setIsOpen(true)} />
      <CommandPalette
        aria-label="Search"
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        inputValue={query}
        onInputChange={setQuery}
      >
        <CommandPalette.Search
          placeholder="Search pages, resources and actions"
          isLoading={isLoading}
          onClose={() => setIsOpen(false)}
        />
        <CommandPalette.Scopes
          aria-label="Search in"
          selected={scope}
          onSelectedChange={(key) => setScope(String(key))}
        >
          <CommandPalette.Scope id="all">All</CommandPalette.Scope>
          <CommandPalette.Scope id="pages" icon="file-lines">
            Pages
          </CommandPalette.Scope>
          <CommandPalette.Scope id="stores" icon="box-archive">
            Stores
          </CommandPalette.Scope>
        </CommandPalette.Scopes>
        <CommandPalette.Body
          preview={
            <div className="p-4">
              <Typography size="sm" className="text-primary">
                {focused ?? "Nothing focused"}
              </Typography>
            </div>
          }
        >
          <CommandPalette.List
            aria-label="Results"
            onFocusKey={setFocused}
            onAction={() => setIsOpen(false)}
            renderEmptyState={() => (
              <CommandPalette.Empty
                title={`No results for “${query}”`}
                description="Try another word or search everything."
              />
            )}
          >
            {sections.map((section) => (
              <CommandPalette.Section
                key={section}
                id={section}
                title={section}
              >
                {visible
                  .filter((item) => item.section === section)
                  .map((item) => (
                    <CommandPalette.Item
                      key={item.id}
                      id={item.id}
                      label={item.label}
                      icon={item.icon}
                      description={
                        "description" in item ? item.description : undefined
                      }
                      meta={
                        <Tag size="sm" radius="sqrt">
                          {item.section}
                        </Tag>
                      }
                    />
                  ))}
              </CommandPalette.Section>
            ))}
          </CommandPalette.List>
        </CommandPalette.Body>
        <CommandPalette.Footer>
          <CommandPalette.Hint keys={["↑", "↓"]} label="Navigate" />
          <CommandPalette.Hint keys={["↵"]} label="Open" />
          <CommandPalette.Hint keys={["Esc"]} label="Close" />
        </CommandPalette.Footer>
      </CommandPalette>
    </div>
  );
}

const meta = {
  title: "design-system/Command Palette",
  component: Template,
  tags: ["autodocs"],
  argTypes: {
    isLoading: { control: "boolean" },
  },
  args: {
    isLoading: false,
  },
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Loading: Story = {
  args: { isLoading: true },
};

export const CompactTrigger: Story = {
  render: () => (
    <div className="w-10">
      <CommandPalette.Trigger isCompact onPress={() => undefined} />
    </div>
  ),
};
