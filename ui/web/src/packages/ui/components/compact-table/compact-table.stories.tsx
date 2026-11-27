import { Meta, StoryObj } from "@storybook/react-vite";

import { CompactTable } from "./compact-table";
import { Icon } from "../icon/icon";

const meta = {
  title: "design-system/Compact Table",
  tags: ["autodocs"],
  component: CompactTable,
  args: {
    ariaLabel: "Restore points per store",
    headers: ["Store", "Restore points", "Size"],
    rows: [
      { name: "s3-eu-west-3", count: "812", size: "642 GB" },
      { name: "nas-paris", count: "455", size: "301 GB" },
      { name: "glacier-archive", count: "97", size: "88.4 GB" },
    ].map((row) => ({
      key: row.name,
      cells: [
        <span className="flex items-center gap-2">
          <Icon type="box-archive" size="sm" className="text-tertiary" />
          {row.name}
        </span>,
        row.count,
        row.size,
      ],
    })),
  },
} satisfies Meta<typeof CompactTable>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Scrolling: Story = {
  render: (args) => (
    <div className="h-24">
      <CompactTable {...args} />
    </div>
  ),
};
