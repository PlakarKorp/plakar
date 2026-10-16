import { Meta, StoryObj } from "@storybook/react-vite";

import { ExpandableList } from "./expandable-list";

const meta = {
  title: "design-system/ExpandableList",
  tags: ["autodocs"],
  component: ExpandableList,
  argTypes: {
    items: {
      control: { type: "object" },
    },
  },
} satisfies Meta<typeof ExpandableList>;

export default meta;
type Story = StoryObj<typeof meta>;

export const ManyItems: Story = {
  args: {
    items: [
      {
        icon: "puzzle-piece",
        value: "s3://s3.eu-west-3.amazonaws.com/plakar-backups",
        subValue: "~/data/Korpus/github.com/OpenBSD/www/papers",
      },
      {
        icon: "s3",
        value:
          "s3://s3.eu-west-3.amazonaws.com/plakar-backups/too-long-path-that-should-be-truncated",
        subValue: "~/data/Korpus/github.com/OpenBSD/www/papers",
      },
      {
        icon: "gcs",
        value: "s3://s3.eu-west-3.amazonaws.com/plakar-backups",
        subValue: "~/data/Korpus/github.com/OpenBSD/www/papers",
      },
      {
        icon: "sftp",
        value: "s3://s3.eu-west-3.amazonaws.com/plakar-backups",
        subValue: "~/data/Korpus/github.com/OpenBSD/www/papers",
      },
    ],
  },
};

export const SingleItem: Story = {
  args: {
    items: [
      {
        icon: "sftp",
        value: "s3://s3.eu-west-3.amazonaws.com/plakar-backups",
        subValue: "~/data/Korpus/github.com/OpenBSD/www/papers",
      },
    ],
  },
};
