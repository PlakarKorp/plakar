import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Button } from "..";
import { Drawer } from "./drawer";

type TempalteProps = {
  children?: React.ReactNode;
  title: string;
};

const Template = ({ children, title }: TempalteProps) => {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onPress={() => setOpen(true)}>Open modal</Button>
      <Drawer open={open} setOpen={setOpen} title={title}>
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
