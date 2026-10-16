import type { Meta, StoryObj } from "@storybook/react-vite";
import clsx from "clsx";
import { useState, type ComponentProps } from "react";
import { Key, useFilter } from "react-aria-components";

import { MultiSelect } from "./multi-select";

type StoryArgs = ComponentProps<typeof MultiSelect> & {
  selectOptions: string[];
};

const meta: Meta<StoryArgs> = {
  title: "design-system/Multi Select",
  component: MultiSelect,
  tags: ["autodocs"],
  argTypes: {
    selectOptions: {
      control: { type: "object" },
    },
  },
  args: {
    label: "Select user roles",
    supportText: "You can select multiple roles",
    selectOptions: ["Admin", "Manager", "Member", "Contributor", "Guest"],
    isDisabled: false,
    isInvalid: false,
    isPending: false,
  },
  render: ({ selectOptions, ...rest }) => (
    <div className={clsx("m-auto w-100")}>
      <MultiSelect {...rest}>
        <MultiSelect.ListBox>
          {selectOptions.map((name) => (
            <MultiSelect.Item key={name} id={name} name={name} />
          ))}
        </MultiSelect.ListBox>
      </MultiSelect>
    </div>
  ),
};

export default meta;

type Story = StoryObj<StoryArgs>;

export const Default: Story = {};

export const Disabled: Story = {
  args: {
    defaultValue: ["Member", "Guest"],
    isDisabled: true,
  },
};

export const Invalid: Story = {
  args: {
    isInvalid: true,
  },
};

export const Pending: Story = {
  args: {
    defaultValue: ["Member", "Guest"],
    isPending: true,
  },
};

function ControlledExample() {
  const [value, setValue] = useState<Key[]>(["Manager", "Contributor"]);

  return (
    <div
      className={clsx("flex w-full flex-col items-center justify-center gap-4")}
    >
      <MultiSelect value={value} onChange={setValue}>
        <MultiSelect.ListBox>
          {["Admin", "Manager", "Member", "Contributor", "Guest"].map(
            (name) => (
              <MultiSelect.Item key={name} id={name} name={name} />
            )
          )}
        </MultiSelect.ListBox>
      </MultiSelect>

      <div className="text-primary">Selected values: {value.join(", ")}</div>
    </div>
  );
}

export const Controlled: Story = {
  render: ControlledExample,
};

const SearchableExample = ({ selectOptions, ...rest }: StoryArgs) => {
  const { contains } = useFilter({ sensitivity: "base" });

  return (
    <div className={clsx("m-auto w-100")}>
      <MultiSelect {...rest}>
        <MultiSelect.SearchableListBox filter={contains}>
          {selectOptions.map((name) => (
            <MultiSelect.Item key={name} id={name} name={name} />
          ))}
        </MultiSelect.SearchableListBox>
      </MultiSelect>
    </div>
  );
};

export const Searchable: Story = {
  render: SearchableExample,
};
