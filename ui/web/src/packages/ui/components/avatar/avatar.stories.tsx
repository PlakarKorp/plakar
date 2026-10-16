import type { Meta, StoryObj } from "@storybook/react-vite";

import { Avatar } from "./avatar";
import { AvatarShape, AvatarSize, AvatarVariant } from "./avatar.constants";

const meta = {
  title: "design-system/Avatar",
  tags: ["autodocs"],
  component: Avatar,
  argTypes: {
    size: {
      control: "inline-radio",
      options: AvatarSize,
    },
    variant: {
      control: "inline-radio",
      options: AvatarVariant,
    },
  },
} satisfies Meta<typeof Avatar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const TextOnly: Story = {
  args: {
    text: "JC",
  },
};

export const Image: Story = {
  args: {
    text: "JC",
    url: "https://avatars.githubusercontent.com/u/795841?v=4",
  },
};

export const ImageWithInvalidLink: Story = {
  args: {
    text: "JC",
    url: "https://doesnotexist.com/image.jpg",
  },
};

export const Empty: Story = {
  args: {
    text: "",
    url: "",
  },
};

export const Loading: Story = {
  args: {
    text: "JC",
    isLoading: true,
  },
};

export const Variants: Story = {
  args: {
    text: "JC",
  },
  parameters: {
    controls: { disable: true },
  },
  render: (args) => (
    <div className="flex items-center gap-4">
      {AvatarVariant.map((variant) => (
        <div key={variant} className="flex items-center gap-2">
          <Avatar {...args} variant={variant} />
          <span className="text-secondary text-sm">{variant}</span>
        </div>
      ))}
    </div>
  ),
};

export const Sizes: Story = {
  args: {
    text: "JC",
  },
  parameters: {
    controls: { disable: true },
  },
  render: (args) => (
    <div className="flex items-center gap-4">
      {AvatarSize.map((size) => (
        <div key={size} className="flex items-center gap-2">
          <Avatar {...args} size={size} />
          <Avatar icon="user-group" size={size} />
          <Avatar text="" size={size} />
          <span className="text-secondary text-sm">{size}</span>
        </div>
      ))}
    </div>
  ),
};

export const Shapes: Story = {
  args: {
    text: "Root organization",
  },
  parameters: {
    controls: { disable: true },
  },
  render: (args) => (
    <div className="flex items-center gap-4">
      {AvatarShape.map((shape) => (
        <div key={shape} className="flex items-center gap-2">
          <Avatar {...args} shape={shape} variant="group" />
          <span className="text-secondary text-sm">{shape}</span>
        </div>
      ))}
    </div>
  ),
};
