import { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Slider } from "./slider";

const meta = {
  title: "design-system/Slider",
  tags: ["autodocs"],
  component: Slider,
  render: function Render(args) {
    const [value, setValue] = useState(args.value);
    return (
      <Slider {...args} value={value} onChange={setValue} className="w-64" />
    );
  },
} satisfies Meta<typeof Slider>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  args: {
    "aria-label": "Volume",
    value: 40,
    onChange: () => {},
  },
};

export const Fractional: Story = {
  args: {
    "aria-label": "Seek",
    value: 0.3,
    minValue: 0,
    maxValue: 1,
    step: 0.05,
    onChange: () => {},
  },
};

export const Disabled: Story = {
  args: {
    "aria-label": "Volume",
    value: 60,
    isDisabled: true,
    onChange: () => {},
  },
};
