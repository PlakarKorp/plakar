import type { Meta, StoryObj } from "@storybook/react-vite";

import { Breadcrumbs } from "./breadcrumbs";

type Crumb = {
  label: string;
  monospace?: boolean;
};

function Template({ items }: { items: Crumb[] }) {
  return (
    <Breadcrumbs>
      {items.map((item, index) => (
        <Breadcrumbs.Crumb key={index} to="#" monospace={item.monospace}>
          {item.label}
        </Breadcrumbs.Crumb>
      ))}
    </Breadcrumbs>
  );
}

const meta = {
  title: "Components/Breadcrumbs",
  tags: ["autodocs"],
  component: Template,
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        label: "Home",
      },
      {
        label: "Snapshots",
      },
      {
        label: "9b4708e0",
        monospace: true,
      },
    ],
  },
};
