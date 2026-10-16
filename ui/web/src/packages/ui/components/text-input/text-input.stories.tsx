import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { TextInput } from "./text-input";
import { FieldButton } from "../field/field-button";
import { IconTypes } from "../icon/icon.constants";

const meta = {
  title: "design-system/Text Input",
  tags: ["autodocs"],
  args: {
    icon: "bars",
    label: "Label",
    placeholder: "Placeholder text",
    supportText: "Support text",
    className: "max-w-80",
    isDisabled: false,
    isInvalid: false,
    isRequired: false,
  },
  argTypes: {
    icon: {
      control: { type: "select" },
      options: [undefined, ...IconTypes],
      table: {
        type: { summary: "IconType" },
      },
    },
  },
  component: TextInput,
} satisfies Meta<typeof TextInput>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Disabled: Story = {
  args: {
    isDisabled: true,
  },
};

export const Invalid: Story = {
  args: {
    isInvalid: true,
  },
};

export const ReadOnly: Story = {
  args: {
    value: "Read-only value",
    isReadOnly: true,
  },
};

export const Revealable: Story = {
  args: {
    label: "Password",
    type: "password",
  },
  render: (args) => {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    const [value, setValue] = useState("correct horse battery staple");
    return <TextInput {...args} value={value} onChange={setValue} />;
  },
};

export const WithClearButton: Story = {
  args: {
    supportText: "Start typing to see the clear button",
  },
  render: (args) => {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    const [localFilename, setLocalFilename] = useState("");
    return (
      <TextInput
        {...args}
        onChange={(value) => {
          setLocalFilename(value);
        }}
        value={localFilename}
        suffix={
          localFilename ? (
            <FieldButton
              icon="close"
              aria-label="Clear text"
              onPress={() => setLocalFilename("")}
              className="-mr-2"
            />
          ) : undefined
        }
      />
    );
  },
};
