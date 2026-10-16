import { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps, useState } from "react";

import { Filters } from "./filters";

function Template(args: ComponentProps<typeof Filters>) {
  const [defaultSelected, setDefaultSelected] = useState<{
    mime: string[] | null;
    animal: string[] | null;
  }>({
    mime: null,
    animal: null,
  });

  return (
    <Filters
      {...args}
      defaultSelected={defaultSelected}
      onApply={(selected) => {
        setDefaultSelected({
          mime: selected.mime || null,
          animal: selected.animal || null,
        });
      }}
    />
  );
}

const meta = {
  title: "design-system/Filters",
  component: Filters,
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        id: "mime",
        label: "Mime",
        options: [
          {
            id: "images",
            icon: "md-image",
            name: "Images",
          },
          {
            id: "pdfs",
            icon: "md-picture-as-pdf",
            name: "PDFs",
          },
          {
            id: "videos",
            icon: "md-video-file",
            name: "Videos",
          },
          {
            id: "audio",
            icon: "md-audio-file",
            name: "Audio",
          },
          {
            id: "archives",
            icon: "md-folder-zip",
            name: "Archives",
          },
          {
            id: "text",
            icon: "md-file-copy",
            name: "Text files",
          },
        ],
      },
      {
        id: "animal",
        label: "Animal",
        options: [
          { id: "chipmunk", name: "Chipmunk" },
          { id: "kitten", name: "Kitten" },
          { id: "puppy", name: "Puppy" },
          { id: "rabbit", name: "Rabbit" },
          { id: "squirrel", name: "Squirrel" },
        ],
        isMulti: true,
      },
    ],
    defaultSelected: {
      mime: null,
      animal: null,
    },
    onApply: (selected) => {
      console.log("Selected filters:", selected);
    },
  },
  render: (args) => <Template {...args} />,
};
