import { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";
import { fn } from "storybook/test";

import { IntegrationCard } from "./integration-card";
import { IntegrationCardSkeleton } from "./integration-card-skeleton";
import { IntegrationInstallationStatuses } from "./integration-card.constants";

type TemplateProps = Omit<ComponentProps<typeof IntegrationCard>, "types"> & {
  isStorage: boolean;
  isSource: boolean;
  isDestination: boolean;
  isSecretProvider: boolean;
  isInventory: boolean;
};

function Template({
  isStorage,
  isSource,
  isDestination,
  isSecretProvider,
  isInventory,
  ...props
}: TemplateProps) {
  const types = {
    storage: isStorage,
    source: isSource,
    destination: isDestination,
    secret_provider: isSecretProvider,
    inventory: isInventory,
  };

  return <IntegrationCard types={types} {...props} />;
}

const meta = {
  title: "components/Integration Card",
  tags: ["autodocs"],
  component: Template,
  argTypes: {
    installationStatus: {
      control: { type: "radio" },
      options: IntegrationInstallationStatuses,
    },
    documentationProps: {
      control: false,
      table: { disable: true },
    },
    onInstall: {
      control: false,
      table: { disable: true },
    },
    onRemove: {
      control: false,
      table: { disable: true },
    },
    stage: {
      control: { type: "radio" },
      options: ["stable", "testing", "beta", "devel", "unknown"],
    },
  },
} satisfies Meta<typeof Template>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    name: "k8s",
    displayName: "Integration name",
    version: "v1.0.2",
    stage: "stable",
    installationStatus: "installed",
    isStorage: true,
    isSource: true,
    isDestination: true,
    isSecretProvider: false,
    isInventory: false,

    tags: ["Lorem", "Ipsum", "Dolor"],

    description:
      "Lorem ipsum dolor sit amet consectetur. Mauris leo vel elementum volutpat. Sed eget pulvinar suspendisse dignissim. Aliquam urna placerat cras pellentesque integer. Tristique non magna convallis nec.",
    documentationProps: { onPress: fn() },
    pending: false,
    onInstall: fn(),
    onRemove: fn(),
  },
};

/** What somebody who may read the catalogue and not install from it sees. */
export const NotPermitted: Story = {
  args: {
    ...Default.args,
    installationStatus: "not-installed",
    onUpgrade: fn(),
    latestVersion: "v1.1.0",
    isDisabled: true,
  },
};

export const Loading = {
  argTypes: {},
  render: () => <IntegrationCardSkeleton />,
};
