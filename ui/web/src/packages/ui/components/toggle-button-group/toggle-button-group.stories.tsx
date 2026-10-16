import type { Meta, StoryObj } from "@storybook/react-vite";

import { ToggleButtonGroup } from "./toggle-button-group";

const meta: Meta<typeof ToggleButtonGroup> = {
  title: "design-system/Toggle Button Group",
  tags: ["autodocs"],
  component: ToggleButtonGroup,
};

export default meta;

type Story = StoryObj<typeof ToggleButtonGroup>;

export const Default: Story = {
  args: {
    selectionMode: "single",
    isAnimated: true,
  },
  argTypes: {
    selectionMode: {
      control: { type: "inline-radio" },
      options: ["single", "multiple"],
    },
  },
  render: (args) => (
    <ToggleButtonGroup {...args} aria-label="Default toggle">
      <ToggleButtonGroup.Button id="a">Compute</ToggleButtonGroup.Button>
      <ToggleButtonGroup.Button id="b" tag="12">
        Block Storage
      </ToggleButtonGroup.Button>
      <ToggleButtonGroup.Button id="c" isDisabled>
        Database
      </ToggleButtonGroup.Button>
      <ToggleButtonGroup.Button id="d">Unknown</ToggleButtonGroup.Button>
    </ToggleButtonGroup>
  ),
};
