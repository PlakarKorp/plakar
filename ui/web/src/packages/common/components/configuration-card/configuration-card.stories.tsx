import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { ConfigurationCard } from "./configuration-card";
import { IconBadge } from "../icon-badge/icon-badge";

function Template(props: ComponentProps<typeof ConfigurationCard.Content>) {
  return <ConfigurationCard.Content {...props} />;
}

const meta = {
  title: "Components/Configuration Card",
  tags: ["autodocs"],
  args: {
    id: "1f9d7122-8e3a-4169-8cff-ae5b36bd2117",
    name: "S3 storage",
    integration: "S3",
    location: "s3://backup.example.com/plakar",
  },
  component: Template,
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (args) => (
    <ConfigurationCard>
      <ConfigurationCard.Content {...args} />
    </ConfigurationCard>
  ),
};

export const ContentOnly: Story = {};

export const AllCardsWithIcon: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: () => (
    <div className="grid grid-cols-2 gap-4">
      <ConfigurationCard>
        <IconBadge icon="upload" />
        <ConfigurationCard.Content
          id="1f9d7122-8e3a-4169-8cff-ae5b36bd2117"
          name="S3 Source"
          integration="S3"
          location="s3://data.example.com/plakar"
        />
      </ConfigurationCard>
      <ConfigurationCard>
        <IconBadge icon="download" />
        <ConfigurationCard.Content
          id="2a8d7122-4b3a-4269-9cff-be6b36bd3228"
          name="GCS Destination"
          integration="Google Cloud Storage"
          location="gs://data-backup/plakar"
        />
      </ConfigurationCard>
      <ConfigurationCard>
        <IconBadge icon="box-archive" />
        <ConfigurationCard.Content
          id="3b7d8122-5c4b-4369-0dff-ce7c36bd4339"
          name="Azure Blob Store"
          integration="Azure Blob Storage"
          location="https://myaccount.blob.core.windows.net/plakar"
        />
      </ConfigurationCard>
      <ConfigurationCard>
        <IconBadge icon="lock" />
        <ConfigurationCard.Content
          id="4c6d9122-6d5c-4469-1eff-de8c36bd5440"
          name="Vault Secrets Provider"
          integration="HashiCorp Vault"
          location="https://vault.example.com:8200/v1/secret/data/plakar"
        />
      </ConfigurationCard>
    </div>
  ),
};
