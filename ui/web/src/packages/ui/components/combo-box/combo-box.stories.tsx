import type { Meta, StoryObj } from "@storybook/react-vite";
import type { ComponentProps } from "react";
import { Collection } from "react-aria-components";
import { useAsyncList } from "react-stately";

import { ComboBox, ComboBoxOption } from "./combo-box";

type StoryArgs = ComponentProps<typeof ComboBox> & {
  selectOptions: string[];
};

const meta: Meta<StoryArgs> = {
  title: "design-system/Combo Box",
  component: ComboBox,
  tags: ["autodocs"],
  argTypes: {
    selectOptions: {
      control: { type: "object" },
    },
    children: { table: { disable: true } },
  },
  args: {
    label: "Label",
    placeholder: "Select an animal",
    selectOptions: ["Chipmunk", "Kitten", "Puppy", "Rabbit", "Squirrel"],
    supportText: "Support text",
    className: "w-[200px]",
    onSelectionChange: (value) => console.log("Selected:", value),
    isRequired: false,
  },
  render: ({ selectOptions, ...rest }) => (
    <ComboBox {...rest}>
      {selectOptions.map((name) => (
        <ComboBox.Item key={name} id={name} name={name} />
      ))}
    </ComboBox>
  ),
};

export default meta;

type Story = StoryObj<StoryArgs>;

export const Default: Story = {};

export const Disabled: Story = {
  args: {
    isDisabled: true,
  },
};

export const Invalid: Story = {
  args: {
    isInvalid: true,
  },
};

export const ReadOnly: Story = {
  args: {
    selectedKey: "Kitten",
    isReadOnly: true,
  },
};

function RenderAsyncLoading(args: StoryArgs) {
  const list = useAsyncList<ComboBoxOption>({
    async load({ signal, cursor, filterText }) {
      const res = await fetch(
        cursor || `https://swapi.py4e.com/api/people/?search=${filterText}`,
        { signal }
      );
      const json = await res.json();

      return {
        items: json.results.map((item: { name: string }) => ({
          id: item.name,
          name: item.name,
        })),
        cursor: json.next,
      };
    },
  });

  return (
    <ComboBox
      {...args}
      label="Star Wars Characters"
      placeholder={undefined}
      inputValue={list.filterText}
      onInputChange={list.setFilterText}
      supportText="Data loaded asynchronously"
    >
      <Collection items={list.items}>
        {(item) => (
          <ComboBox.Item key={item.id} id={item.id} name={item.name} />
        )}
      </Collection>
      <ComboBox.Loading
        onLoadMore={list.loadMore}
        isLoading={list.loadingState === "loadingMore"}
      />
    </ComboBox>
  );
}

export const AsyncLoading: Story = {
  parameters: {
    controls: {
      exclude: [
        "label",
        "selectOptions",
        "placeholder",
        "supportText",
        "onSelectionChange",
      ],
    },
  },
  render: (args) => <RenderAsyncLoading {...args} />,
};
