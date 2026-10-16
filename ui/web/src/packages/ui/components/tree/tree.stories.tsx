import { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { Key } from "react-aria-components";

import { Tree } from "./tree";
import { Tag } from "../tag/tag";
import { Typography } from "../typography/typography";

const meta = {
  title: "design-system/Tree",
  tags: ["autodocs"],
  component: Tree,
} satisfies Meta<typeof Tree>;

export default meta;
type Story = StoryObj<typeof meta>;

function DefaultLabel({
  label,
  integration,
}: {
  label: string;
  integration: string;
}) {
  return (
    <div className="flex items-center gap-1.5">
      <Tag variant="source" size="sm" radius="sqrt">
        {integration}
      </Tag>
      {label}
    </div>
  );
}

const demoItems = [
  {
    id: "1",
    label: "Plakar-staging-data",
    description: "Object Storage - S3",
    children: [
      {
        id: "1-1",
        label: "AWS staging data S3 Source",
        content: (
          <DefaultLabel label="AWS staging data S3 Source" integration="s3" />
        ),
      },
      {
        id: "1-2",
        label: "plakar-staging-data-1-2",
        content: (
          <DefaultLabel label="plakar-staging-data-1-2" integration="imap" />
        ),
      },
    ],
  },
  {
    id: "2",
    label: "Webserver",
    description: "Compute",
    children: [
      {
        id: "2-1",
        label: "webserver-2-1",
        content: <DefaultLabel label="webserver-2-1" integration="sftp" />,
      },
      {
        id: "2-2",
        label: "webserver-2-2",
        content: <DefaultLabel label="webserver-2-2" integration="api" />,
      },
      {
        id: "2-3",
        label: "webserver-2-3",
        content: <DefaultLabel label="webserver-2-3" integration="proxmox" />,
      },
    ],
  },
];

export const Default: Story = {
  args: {
    items: demoItems,
  },
};

export const MultipleLevels: Story = {
  args: {
    items: [
      {
        id: "1",
        label: "plakar-staging-data",
        children: [
          {
            id: "1-1",
            label: "AWS staging data S3 Source",
            children: [
              { id: "1-1-1", label: "Child 1" },
              {
                id: "1-1-2",
                label: "Child 2",
                children: [
                  { id: "1-1-2-1", label: "Grandchild 1" },
                  { id: "1-1-2-2", label: "Grandchild 2" },
                  {
                    id: "1-1-2-3",
                    label: "Grandchild 3",
                    children: [
                      { id: "1-1-2-3-1", label: "Great Grandchild 1" },
                      { id: "1-1-2-3-2", label: "Great Grandchild 2" },
                    ],
                  },
                  { id: "1-1-2-4", label: "Grandchild 4" },
                ],
              },
            ],
          },
          { id: "1-2", label: "plakar-staging-data-1-2" },
        ],
      },
    ],
  },
};

function ControlledTreeStory(args: React.ComponentProps<typeof Tree>) {
  const [selectedKeys, setSelectedKeys] = useState<Set<Key>>(
    new Set(["1-1", "1-2", "2-2"])
  );

  return (
    <div className="space-y-8">
      <Tree
        {...args}
        selectedKeys={selectedKeys}
        onSelectionChange={setSelectedKeys}
      />

      <div className="border-secondary rounded-lg border p-3">
        <Typography weight="medium" size="sm">
          Selected keys
        </Typography>
        <Typography as="div" size="sm" className="text-tertiary mt-2">
          {selectedKeys.size > 0
            ? Array.from(selectedKeys).join(", ")
            : "No selection"}
        </Typography>
      </div>
    </div>
  );
}

export const Controlled: Story = {
  render: (args) => <ControlledTreeStory {...args} />,
  args: {
    items: demoItems,
  },
};
