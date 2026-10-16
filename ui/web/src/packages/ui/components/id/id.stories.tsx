import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { Id } from "./id";
import { TypographySizes } from "../typography/typography.constants";

type IdStoryArgs = ComponentProps<typeof Id>;

const meta = {
  title: "design-system/Id",
  tags: ["autodocs"],
  component: Id,
  argTypes: {
    value: {
      control: "text",
      table: { type: { summary: "string" } },
    },
    helpText: {
      control: "text",
      table: { type: { summary: "string" } },
    },
    size: {
      control: "select",
      options: TypographySizes,
      table: { type: { summary: "TypographySize" } },
    },
  },
} satisfies Meta<IdStoryArgs>;

export default meta;

type Story = StoryObj<IdStoryArgs>;

export const Default: Story = {
  args: {
    value: "720de028-0000-0000-0000-000000000000",
    noTruncate: false,
  },
};
