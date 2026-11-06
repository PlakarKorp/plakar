import type { Meta, StoryObj } from "@storybook/react-vite";

import { Callout } from "./callout";
import { Button } from "../button/button";

const meta = {
  title: "design-system/Callout",
  tags: ["autodocs"],
  component: Callout,
  args: {
    icon: "cube",
    title: "Restore point ready",
    description: "12,480 items · 3.2 GB",
  },
} satisfies Meta<typeof Callout>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    action: <Button variant="primary">Browse snapshot</Button>,
  },
};

export const WithoutAction: Story = {};
