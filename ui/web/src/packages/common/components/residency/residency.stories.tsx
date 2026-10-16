import { TagSize } from "@plakar-ui/ui/components";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { ResidencyLabel, ResidencyTag } from "./residency";

const COUNTRIES = [
  { alpha2: "FR", flagEmoji: "🇫🇷", name: "France" },
  { alpha2: "DE", flagEmoji: "🇩🇪", name: "Germany" },
  { alpha2: "US", flagEmoji: "🇺🇸", name: "United States" },
  { alpha2: "NL", flagEmoji: "🇳🇱", name: "Netherlands" },
];

const meta = {
  title: "Components/Residency Tag",
  tags: ["autodocs"],
  args: {
    residency: COUNTRIES[0],
    size: "sm",
    label: "code",
  },
  argTypes: {
    size: {
      control: "select",
      options: TagSize,
    },
    label: {
      control: "inline-radio",
      options: ["code", "name"],
    },
  },
  component: ResidencyTag,
} satisfies Meta<typeof ResidencyTag>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithName: Story = {
  args: {
    label: "name",
  },
};

export const Sizes: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: (args) => (
    <div className="flex items-center gap-4">
      {TagSize.map((size) => (
        <ResidencyTag key={size} {...args} size={size} />
      ))}
    </div>
  ),
};

export const Countries: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: (args) => (
    <div className="flex flex-wrap items-center gap-2">
      {COUNTRIES.map((residency) => (
        <ResidencyTag key={residency.alpha2} {...args} residency={residency} />
      ))}
    </div>
  ),
};

export const WithoutTooltip: Story = {
  args: {
    hasTooltip: false,
  },
};

export const Undeclared: Story = {
  args: {
    residency: { alpha2: null, flagEmoji: null, name: null },
  },
};

export const WithFallback: Story = {
  args: {
    residency: { alpha2: null, flagEmoji: null, name: null },
    fallback: "Undeclared",
  },
};

export const WithHint: Story = {
  args: {
    hint: 'Residency mismatch: this resource is configured for "FR", but its detected residency is "US".',
  },
};

export const Label: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: (args) => (
    <div className="flex items-center gap-4">
      {COUNTRIES.map((residency) => (
        <ResidencyLabel
          key={residency.alpha2}
          {...args}
          residency={residency}
        />
      ))}
    </div>
  ),
};
