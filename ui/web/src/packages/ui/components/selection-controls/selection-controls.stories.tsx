import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { Button } from "..";
import { SelectionControls } from "./selection-controls";

type TemplateProps = {
  text: string;
  altText: string;
  numberOfButtons: number;
  label: string;
  onPress: () => void;
};

function Template({
  text,
  altText,
  numberOfButtons,
  label,
  onPress,
}: TemplateProps) {
  return (
    <SelectionControls text={text} altText={altText} onUnselect={onPress}>
      {Array.from({ length: numberOfButtons }, (_, index) => (
        <Button key={index} variant="primary" onPress={onPress}>
          {label}
        </Button>
      ))}
    </SelectionControls>
  );
}

const meta = {
  title: "design-system/Selection Controls",
  tags: ["autodocs"],
  component: Template,
  argTypes: {
    onPress: {
      table: {
        disable: true,
      },
    },
  },
  args: {
    onPress: fn(),
  },
} satisfies Meta<TemplateProps>;

export default meta;

type Story = StoryObj<TemplateProps>;

export const Default: Story = {
  args: {
    text: "3 selected",
    altText: "1.2 MB",
    numberOfButtons: 3,
    label: "Label",
  },
};
