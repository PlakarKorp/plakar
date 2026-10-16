import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { CopyableValue } from "./copyable-value";

type CopyableValueStoryArgs = ComponentProps<typeof CopyableValue>;

const meta = {
  title: "design-system/CopyableValue",
  tags: ["autodocs"],
  component: CopyableValue,
  argTypes: {
    label: {
      control: "text",
      table: { type: { summary: "string" } },
    },
    value: {
      control: "text",
      table: { type: { summary: "string" } },
    },
    copyLabel: {
      control: "text",
      table: { type: { summary: "string" } },
    },
  },
} satisfies Meta<CopyableValueStoryArgs>;

export default meta;

type Story = StoryObj<CopyableValueStoryArgs>;

export const Default: Story = {
  args: {
    label: "Key fingerprint",
    value:
      "SHA256:3f9a1c4e7b2d8056a1f3c9e4b7d20a8f6c1e5b3d9a7f2c4e8b1d6a0f3c9e7b2d",
  },
};

export const Command: Story = {
  args: {
    label: "Verify with",
    copyLabel: "command",
    value: "plakman attestation-verify --key residency.pub residency.jsonl",
  },
};

export const PublicKey: Story = {
  args: {
    label: "Public key",
    value:
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIK7Qm4vXJ2pL9sN3rT8wY1cF5hB6dE0aG2jU4mZ9xV7n attestation@plakman",
  },
};
