import { IconSize, IconSizes } from "@plakar-ui/ui/components";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { FileIcon } from "./file-icon";

const meta: Meta<typeof FileIcon> = {
  title: "components/File Icon",
  tags: ["autodocs"],
  argTypes: {
    contentType: {
      control: { type: "select" },
      options: [
        undefined,
        "images",
        "pdfs",
        "videos",
        "audio",
        "archives",
        "text",
      ],
    },
    size: {
      control: { type: "select" },
      options: [undefined, ...IconSizes],
    },
  },
  component: FileIcon,
};

export default meta;

type Story = StoryObj<typeof FileIcon>;

export const Default: Story = {};

export const All: Story = {
  render: () => {
    const types: Array<{ label: string; contentType?: string }> = [
      { label: "image/*", contentType: "image/png" },
      { label: "video/*", contentType: "video/mp4" },
      { label: "audio/*", contentType: "audio/mpeg" },
      { label: "application/pdf", contentType: "application/pdf" },
      { label: "application/zip", contentType: "application/zip" },
      { label: "text/plain (fallback)", contentType: "text/plain" },
      { label: "undefined (fallback)", contentType: undefined },
    ];

    return (
      <div className="grid gap-4">
        {types.map((t) => (
          <div key={t.label} className="grid gap-2">
            <div>{t.label}</div>
            <div className="flex flex-wrap items-center gap-3">
              {(IconSizes as readonly string[]).map((s) => (
                <div
                  key={`${t.label}-${s ?? "default"}`}
                  className="grid w-14 justify-center"
                >
                  <FileIcon contentType={t.contentType} size={s as IconSize} />
                  <div className="mt-1 text-center text-xs">
                    {s ?? "default"}
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    );
  },
};
