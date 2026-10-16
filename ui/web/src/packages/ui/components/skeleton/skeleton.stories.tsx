import type { Meta, StoryObj } from "@storybook/react-vite";

import { IconSizes } from "..";
import { Skeleton } from "./skeleton";
import {
  TypographySizes,
  TypographyVariants,
} from "../typography/typography.constants";

const meta = {
  title: "design-system/Skeleton",
  tags: ["autodocs"],
  component: Skeleton,
  argTypes: {
    variant: {
      control: false,
      table: { disable: true },
    },
  },
} satisfies Meta<typeof Skeleton>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Text: Story = {
  argTypes: {
    type: {
      control: { type: "radio" },
      options: TypographyVariants,
    },
    size: {
      control: { type: "radio" },
      options: TypographySizes,
    },
  },
  args: {
    variant: "text",
    type: "body",
  },
};

export const Icon: Story = {
  argTypes: {
    size: {
      control: { type: "radio" },
      options: IconSizes,
    },
  },
  args: {
    variant: "icon",
    size: "lg",
  },
};

export const Circle: Story = {
  argTypes: {
    width: {
      control: { type: "number" },
    },
  },
  args: {
    variant: "circle",
    width: 200,
  },
};

export const Rectangle: Story = {
  argTypes: {
    width: {
      control: { type: "number" },
    },
    height: {
      control: { type: "number" },
    },
  },
  args: {
    variant: "rectangle",
    width: 700,
    height: 200,
  },
};
