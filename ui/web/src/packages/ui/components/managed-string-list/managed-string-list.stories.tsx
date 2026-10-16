import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { ManagedStringList } from "./managed-string-list";

const meta = {
  title: "design-system/ManagedStringList",
  component: ManagedStringList,
  tags: ["autodocs"],
} satisfies Meta<typeof ManagedStringList>;

export default meta;

type Story = StoryObj<typeof meta>;

function RenderDefault() {
  const [items, setItems] = useState(["production", "staging", "dev"]);
  const [isAdding, setIsAdding] = useState(false);

  return (
    <div className="w-80">
      <ManagedStringList>
        <ManagedStringList.AddForm
          placeholder="Add an environment…"
          isAdding={isAdding}
          onAdd={(value) => {
            setIsAdding(true);
            setTimeout(() => {
              setItems((prev) => [...prev, value]);
              setIsAdding(false);
            }, 800);
          }}
        />
        <ManagedStringList.Collection isEmpty={items.length === 0}>
          {items.map((item) => (
            <ManagedStringList.Item
              key={item}
              label={item}
              onRemove={() =>
                new Promise<void>((resolve) =>
                  setTimeout(() => {
                    setItems((prev) => prev.filter((i) => i !== item));
                    resolve();
                  }, 800)
                )
              }
            />
          ))}
        </ManagedStringList.Collection>
      </ManagedStringList>
    </div>
  );
}

export const Default: Story = {
  render: () => <RenderDefault />,
};

const RESOURCES: Record<string, string[]> = {
  production: ["db-prod", "cache-prod", "api-prod"],
  staging: ["db-staging"],
  dev: [],
};

const COUNTS: Record<string, number> = {
  production: 3,
  staging: 1,
  dev: 0,
};

function RenderExpandable() {
  const [items, setItems] = useState(["production", "staging", "dev"]);

  return (
    <div className="w-96">
      <ManagedStringList>
        <ManagedStringList.AddForm
          placeholder="Add an environment…"
          onAdd={(value) => setItems((prev) => [...prev, value])}
        />
        <ManagedStringList.Collection isEmpty={items.length === 0}>
          {items.map((item) => {
            const count = COUNTS[item] ?? 0;
            return (
              <ManagedStringList.Item
                key={item}
                label={item}
                altText={
                  count === 0 ? (
                    <span className="text-quaternary italic">Unused</span>
                  ) : (
                    <>
                      {count} resource{count !== 1 ? "s" : ""}
                    </>
                  )
                }
                onRemove={() =>
                  new Promise<void>((resolve) =>
                    setTimeout(() => {
                      setItems((prev) => prev.filter((i) => i !== item));
                      resolve();
                    }, 800)
                  )
                }
                confirmRemoveMessage="This environment has resources assigned to it."
                expandedContent={
                  count > 0 ? (
                    <ul className="flex flex-col gap-1 py-2">
                      {(RESOURCES[item] ?? []).map((r) => (
                        <li key={r} className="text-secondary px-2 text-sm">
                          {r}
                        </li>
                      ))}
                    </ul>
                  ) : null
                }
              />
            );
          })}
        </ManagedStringList.Collection>
      </ManagedStringList>
    </div>
  );
}

export const Expandable: Story = {
  render: () => <RenderExpandable />,
};

function RenderReadOnly() {
  const items = ["production", "staging", "dev"];

  return (
    <div className="w-80">
      <ManagedStringList>
        <ManagedStringList.AddForm
          placeholder="Add an environment…"
          isDisabled
          onAdd={() => {}}
        />
        <ManagedStringList.Collection isEmpty={items.length === 0}>
          {items.map((item) => (
            <ManagedStringList.Item
              key={item}
              label={item}
              isDisabled
              onRemove={() => {}}
            />
          ))}
        </ManagedStringList.Collection>
      </ManagedStringList>
    </div>
  );
}

export const ReadOnly: Story = {
  render: () => <RenderReadOnly />,
};
