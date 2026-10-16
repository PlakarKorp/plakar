import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";
import { DialogTrigger } from "react-aria-components";

import { Button } from "..";
import { Modal } from "./modal";

type TemplateProps = ComponentProps<typeof Modal> & {
  title: string;
  description?: string;
  isCustom?: boolean;
  footerChildren?: React.ReactNode;
};

const Template = ({
  title,
  description,
  children,
  footerChildren,
  isCustom,
  ...modalProps
}: TemplateProps) => (
  <DialogTrigger>
    <Button>Open modal</Button>
    <Modal aria-label={title} {...modalProps}>
      {!isCustom ? (
        <Modal.Dialog className="max-w-2xl">
          {title && <Modal.Header title={title} description={description} />}
          <Modal.Content>{children}</Modal.Content>
          <Modal.Footer>{footerChildren}</Modal.Footer>
        </Modal.Dialog>
      ) : (
        children
      )}
    </Modal>
  </DialogTrigger>
);

const meta = {
  title: "design-system/Modal",
  component: Template,
  tags: ["autodocs"],
  argTypes: {
    title: {
      control: "text",
    },
    description: {
      control: "text",
    },
    isDismissable: {
      control: "boolean",
    },
    children: {
      control: "text",
      table: { type: { summary: "React.ReactNode" } },
    },
    isCustom: { table: { disable: true } },
    footerChildren: { table: { disable: true } },
  },
  args: {
    isDismissable: true,
  },
} satisfies Meta<TemplateProps>;

export default meta;

type Story = StoryObj<TemplateProps>;

export const Default: Story = {
  args: {
    title: "Modal title",
    description: "This is a description",
    children:
      "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nunc hendrerit molestie sapien at facilisis. Nullam consectetur rutrum est vel cursus. Suspendisse feugiat nulla nec velit malesuada gravida. In accumsan ex in blandit luctus. Ut sit amet purus finibus, viverra magna eu, fringilla odio. Nullam hendrerit enim a consequat egestas. Aenean tortor eros, euismod non finibus id, laoreet id sem.",
    footerChildren: (
      <Button slot="close" onPress={() => alert("Footer clicked")}>
        Action
      </Button>
    ),
  },
};

export const WithContentOverflow: Story = {
  args: {
    ...Default.args,
    title:
      "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nunc hendrerit molestie sapien at facilisis. Nullam consectetur rutrum est vel cursus.",
    description:
      "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nunc hendrerit molestie sapien at facilisis. Nullam consectetur rutrum est vel cursus.",
    children: (
      <>
        {[...Array(50)].map((_, index) => (
          <p key={index}>
            Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nunc
            hendrerit molestie sapien at facilisis. Nullam consectetur rutrum
            est vel cursus. Suspendisse feugiat nulla nec velit malesuada
            gravida. In accumsan ex in bland it luctus. Ut sit amet purus
            finibus, viverra magna eu, fringilla odio. Nullam hendrerit enim a
            consequat egestas. Aenean tortor eros, euismod non finibus id,
            laoreet id sem.
          </p>
        ))}
      </>
    ),
  },
};

export const Custom: Story = {
  argTypes: {
    title: { table: { disable: true } },
    description: { table: { disable: true } },
    children: { table: { disable: true } },
  },
  args: {
    isCustom: true,
    children: (
      <div className="bg-primary text-primary flex flex-col items-center justify-center gap-4 p-4">
        This is a custom modal content. <Button slot="close">Close</Button>
      </div>
    ),
  },
};
