import { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { Id } from "..";
import { DescriptionList } from "./description-list";
import {
  DescriptionListDirection,
  DescriptionListSize,
} from "./description-list.constants";

const meta: Meta<typeof DescriptionList> = {
  title: "design-system/Description List",
  component: DescriptionList,
  tags: ["autodocs"],
  argTypes: {
    direction: {
      control: "inline-radio",
      options: DescriptionListDirection,
      defaultValue: "col",
    },
    size: {
      control: "inline-radio",
      options: DescriptionListSize,
      defaultValue: "col",
    },
    singleColumn: {
      control: { type: "boolean" },
      defaultValue: false,
      if: { arg: "direction", eq: "row" },
    },
  },
};

export default meta;
type Story = StoryObj<typeof DescriptionList>;

const renderTemplate = (args: ComponentProps<typeof DescriptionList>) => (
  <div className="max-w-4xl">
    <DescriptionList {...args}>
      <DescriptionList.Item
        title="ID"
        description={
          <Id
            value="12862b5b-aa16-4320-b1c7-ddaec6b21095"
            size={args.size}
            weight="medium"
            className="h-5"
          />
        }
      />
      <DescriptionList.Item
        title="Location"
        description="fs:///Users/edouarddecarheil/.plakar
"
      />
      <DescriptionList.Item title="Chunking" description="FASTCDC" />
      <DescriptionList.Item title="Compression" description="LZ4" />
      <DescriptionList.Item
        title="Encryption"
        description="AES256-GCM-SIV
"
      />
    </DescriptionList>
  </div>
);

export const Default: Story = {
  args: { direction: "row", size: "sm", singleColumn: false },
  render: (args) => renderTemplate(args),
};

export const Col: Story = {
  args: { direction: "col" },
  parameters: {
    controls: { disable: true },
  },
  render: (args) => renderTemplate(args),
};

export const Row: Story = {
  args: { direction: "row", size: "sm" },
  parameters: {
    controls: { disable: true },
  },
  render: (args) => renderTemplate(args),
};

export const RowSingleColumn: Story = {
  args: { direction: "row", size: "sm", singleColumn: true },
  parameters: {
    controls: { disable: true },
  },
  render: (args) => renderTemplate(args),
};
