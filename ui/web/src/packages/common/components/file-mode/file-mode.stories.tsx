import type { TypographySize } from "@plakar-ui/ui/components";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { FileMode } from "./file-mode";
import { Mode } from "../../api/enrich";

const MODE_SETUID = 1 << (32 - 1 - 8);
const MODE_SETGID = 1 << (32 - 1 - 9);
const MODE_STICKY = 1 << (32 - 1 - 11);

type StoryArgs = ComponentProps<typeof FileMode> & Args;
type Args = {
  value: string;
  size: TypographySize;
  setuid?: boolean;
  setgid?: boolean;
  sticky?: boolean;
};

const makeMode = ({ value, setuid, setgid, sticky }: Args) => {
  const perms = parseInt(value, 8) & 0o777;
  const flags =
    (setuid ? MODE_SETUID : 0) |
    (setgid ? MODE_SETGID : 0) |
    (sticky ? MODE_STICKY : 0);
  return Mode(perms | flags);
};

const meta: Meta<StoryArgs> = {
  title: "Components/FileMode",
  tags: ["autodocs"],
  component: FileMode,
  argTypes: {
    size: {
      control: { type: "radio" },
      options: ["sm", "md", "lg"],
    },
    value: { control: "text" },
    setuid: { control: "boolean" },
    setgid: { control: "boolean" },
    sticky: { control: "boolean" },
    mode: { control: false, table: { disable: true } },
  },
};
export default meta;

type Story = StoryObj<StoryArgs>;

export const Default: Story = {
  args: {
    value: "0755",
    size: "sm",
    setuid: false,
    setgid: false,
    sticky: false,
  },
  render: (args) => <FileMode mode={makeMode(args)} size={args.size} />,
};

export const Examples: Story = {
  render: () => {
    const rows: Array<Args & { label: string }> = [
      {
        label: "0644",
        value: "0644",
        size: "md",
        setuid: false,
        setgid: false,
        sticky: false,
      },
      {
        label: "0755",
        value: "0755",
        size: "md",
        setuid: false,
        setgid: false,
        sticky: false,
      },
      {
        label: "setuid 0755",
        value: "0755",
        size: "md",
        setuid: true,
        setgid: false,
        sticky: false,
      },
      {
        label: "setgid 0755",
        value: "0755",
        size: "md",
        setuid: false,
        setgid: true,
        sticky: false,
      },
      {
        label: "sticky 1777",
        value: "0777",
        size: "md",
        setuid: false,
        setgid: false,
        sticky: true,
      },
    ];
    return (
      <div style={{ display: "grid", gap: 12 }}>
        {rows.map((r) => (
          <div
            key={r.label}
            style={{ display: "flex", gap: 8, alignItems: "center" }}
          >
            <div style={{ width: 120, opacity: 0.7 }}>{r.label}</div>
            <FileMode mode={makeMode(r)} size={r.size} />
          </div>
        ))}
      </div>
    );
  },
};
