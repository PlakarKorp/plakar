import type { Meta, StoryObj } from "@storybook/react-vite";
import { LinkProps } from "react-aria-components";

import { BackButton } from "./back-button";

function Template({ href }: { href: LinkProps["href"] }) {
  return <BackButton href={href} />;
}

const meta = {
  title: "Components/Back button",
  tags: ["autodocs"],
  component: Template,
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WithLink: Story = {
  args: {
    href: "https://google.com",
  },
};

export const HistoryBack: Story = {
  args: {
    href: undefined,
  },
};
