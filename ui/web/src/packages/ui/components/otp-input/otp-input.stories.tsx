import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { OTPInput } from "./otp-input";

const meta = {
  title: "design-system/OTP Input",
  tags: ["autodocs"],
  args: {
    label: "Authentication code",
    supportText: "Enter the 6-digit code from your authenticator app",
    length: 6,
    value: "",
    onChange: () => {},
    isDisabled: false,
    isInvalid: false,
    isRequired: false,
  },
  component: OTPInput,
  render: (args) => {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    const [code, setCode] = useState(args.value);
    return <OTPInput {...args} value={code} onChange={setCode} />;
  },
} satisfies Meta<typeof OTPInput>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Filled: Story = {
  args: {
    value: "492018",
  },
};

export const Invalid: Story = {
  args: {
    value: "492018",
    isInvalid: true,
    supportText: "That code is incorrect or has expired",
  },
};

export const Disabled: Story = {
  args: {
    value: "4920",
    isDisabled: true,
  },
};

export const FourDigits: Story = {
  args: {
    length: 4,
    supportText: "Enter the 4-digit code",
  },
};
