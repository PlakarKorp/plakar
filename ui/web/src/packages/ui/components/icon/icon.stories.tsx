import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { Icon } from "./icon";
import { IconType, IconTypes } from "./icon.constants";

const INTEGRATION_ICONS: IconType[] = [
  "azblob",
  "etcd",
  "ftp",
  "fs",
  "gcs",
  "imap",
  "k8s",
  "ldap",
  "mongodb",
  "msad",
  "mssql",
  "mysql",
  "nfs",
  "openshift",
  "openstack",
  "openwrt",
  "oracle",
  "postgresql",
  "proxmox",
  "routeros",
  "s3",
  "scaleway",
  "sftp",
  "smb",
  "vault",
  "vmware",
  "vss",
  "webdav",
];

function Template({ size }: ComponentProps<typeof Icon>) {
  return (
    <table className="w-full">
      <tbody>
        {[...IconTypes]
          .sort((a, b) => a.localeCompare(b))
          .map((type) => (
            <tr key={type}>
              <td className="border-tertiary border p-2 text-center">
                <Icon type={type} size={size} />
              </td>
              <td className="border-tertiary border p-2">{type}</td>
            </tr>
          ))}
      </tbody>
    </table>
  );
}

const meta = {
  title: "design-system/Icon",
  tags: ["autodocs"],
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Small: Story = {
  render: () => <Template type="bars" size="sm" />,
};

export const Medium: Story = {
  render: () => <Template type="bars" size="md" />,
};

export const Large: Story = {
  render: () => <Template type="bars" size="lg" />,
};

export const Integrations: Story = {
  render: () => (
    <table className="w-full">
      <tbody>
        {[...INTEGRATION_ICONS]
          .sort((a, b) => a.localeCompare(b))
          .map((type) => (
            <tr key={type}>
              <td className="border-tertiary border p-2 text-center">
                <Icon type={type} size="xl" />
              </td>
              <td className="border-tertiary border p-2">{type}</td>
            </tr>
          ))}
      </tbody>
    </table>
  ),
};
