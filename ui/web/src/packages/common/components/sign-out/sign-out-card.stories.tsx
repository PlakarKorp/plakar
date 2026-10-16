import type { Meta, StoryObj } from "@storybook/react-vite";

import { SignOutCard } from "./sign-out-card";

const meta = {
  title: "Components/Sign Out Card",
  tags: ["autodocs"],
  args: {
    productName: "Plakar",
    account: {
      name: "Édouard de Carheil",
      email: "edouard@plakar.io",
    },
    stayHref: "/",
    isPending: false,
    onSignOut: () => {},
  },
  component: SignOutCard,
  render: (args) => (
    <div className="flex justify-center p-4">
      <SignOutCard {...args} />
    </div>
  ),
} satisfies Meta<typeof SignOutCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithoutAccount: Story = {
  args: {
    account: undefined,
  },
};

export const SigningOut: Story = {
  args: {
    isPending: true,
  },
};

export const Failed: Story = {
  args: {
    error: "The server refused the request. Try again in a moment.",
  },
};
