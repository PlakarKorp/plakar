import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Button } from "..";
import { Drawer } from "./drawer";

type TempalteProps = {
  children?: React.ReactNode;
  title: string;
  footer?: React.ReactNode;
};

const Template = ({ children, title, footer }: TempalteProps) => {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onPress={() => setOpen(true)}>Open modal</Button>
      <Drawer open={open} setOpen={setOpen} title={title} footer={footer}>
        {children}
      </Drawer>
    </>
  );
};

const meta = {
  title: "design-system/Drawer",
  tags: ["autodocs"],
  component: Template,
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof Template>;

export const Default: Story = {
  args: {
    title: "Drawer Title",
    children: "This is the content of the drawer.",
  },
};

export const WithFooter: Story = {
  args: {
    title: "Drawer Title",
    children: (
      <div className="flex flex-col gap-4">
        {Array.from({ length: 30 }, (_, index) => (
          <p key={index}>Line {index + 1} of a content long enough to scroll.</p>
        ))}
      </div>
    ),
    footer: (
      <>
        <Button variant="secondary">Cancel</Button>
        <Button variant="primary">Apply</Button>
      </>
    ),
  },
};
