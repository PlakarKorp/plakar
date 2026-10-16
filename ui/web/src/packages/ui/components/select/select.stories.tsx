import type { Meta, StoryObj } from "@storybook/react-vite";
import clsx from "clsx";
import type { ComponentProps } from "react";
import { useFilter } from "react-aria-components";

import { Button, IconTypes, Typography } from "..";
import { Select } from "./select";

type StoryArgs = ComponentProps<typeof Select> & {
  selectOptions: string[];
};

const meta: Meta<StoryArgs> = {
  title: "design-system/Select",
  component: Select,
  tags: ["autodocs"],
  argTypes: {
    icon: {
      control: { type: "select" },
      options: [undefined, ...IconTypes],
      table: {
        type: { summary: "IconType" },
      },
    },
    selectOptions: {
      control: { type: "object" },
    },
  },
  args: {
    label: "Label",
    placeholder: "Select an animal",
    selectOptions: ["Chipmunk", "Kitten", "Puppy", "Rabbit", "Squirrel"],
    supportText: "Support text",
    className: "w-[200px]",
    onSelectionChange: (value) => console.log("Selected:", value),
    isPending: false,
    isRequired: false,
  },
  render: ({ selectOptions, icon, variant, ...rest }) => (
    <div
      className={clsx(
        "flex w-full items-center justify-center",
        variant === "terminal" && "justify-end bg-slate-900"
      )}
    >
      <Select {...rest} variant={variant}>
        <Select.ListBox>
          {selectOptions.map((name) => (
            <Select.Item key={name} id={name} name={name} icon={icon} />
          ))}
        </Select.ListBox>
      </Select>
    </div>
  ),
};

export default meta;

type Story = StoryObj<StoryArgs>;

export const Default: Story = {};

export const Disabled: Story = {
  args: {
    value: "Kitten",
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
    value: "Kitten",
    isReadOnly: true,
  },
};

export const WithLabelAction: Story = {
  args: {
    labelAction: (
      <Button variant="tertiary" size="sm" leftIcon="arrow-rotate-left">
        Reset
      </Button>
    ),
  },
};

const CustomItem = ({
  name,
  description,
  imageUrl,
}: {
  name: string;
  description: string;
  imageUrl: string;
}) => (
  <div className="flex items-center gap-2">
    <div className="bg-accent-tertiary h-12 w-12 shrink-0 rounded-full">
      <img src={imageUrl} alt={name} className="h-full w-full rounded-full" />
    </div>
    <div className="space-y-0.5">
      <Typography size="sm">{name}</Typography>
      <Typography size="xs" className="text-tertiary line-clamp-2">
        {description}
      </Typography>
    </div>
  </div>
);

export const WithCustomItems: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: () => {
    const options = [
      {
        id: "chipmunk",
        name: "Chipmunk",
        item: (
          <CustomItem
            name="Chipmunk"
            description="Small, striped ground-dwelling rodent with cheek pouches for storing food; quick and chirpy."
            imageUrl="https://upload.wikimedia.org/wikipedia/commons/d/db/Chipmunk_01.jpg"
          />
        ),
      },
      {
        id: "kitten",
        name: "Kitten",
        item: (
          <CustomItem
            name="Kitten"
            description="A juvenile domestic cat; playful, curious, and rapidly developing coordination and social skills."
            imageUrl="https://commons.wikimedia.org/wiki/Special:FilePath/Six%20weeks%20old%20cat%20(aka).jpg"
          />
        ),
      },
      {
        id: "puppy",
        name: "Puppy",
        item: (
          <CustomItem
            name="Puppy"
            description="A young dog; energetic, highly trainable, and in a crucial socialization period."
            imageUrl="https://commons.wikimedia.org/wiki/Special:FilePath/Image-Cavapoo%20puppy.JPG"
          />
        ),
      },
      {
        id: "rabbit",
        name: "Rabbit",
        item: (
          <CustomItem
            name="Rabbit"
            description="Herbivorous lagomorph with long ears and strong hind legs; mostly crepuscular and a quick burrower."
            imageUrl="https://commons.wikimedia.org/wiki/Special:FilePath/Wild%20rabbit.jpg"
          />
        ),
      },
      {
        id: "squirrel",
        name: "Squirrel",
        item: (
          <CustomItem
            name="Squirrel"
            description="Agile tree-dwelling rodent with a bushy tail; often scatters-hoards nuts for later."
            imageUrl="https://commons.wikimedia.org/wiki/Special:FilePath/Eastern%20Grey%20Squirrel.jpg"
          />
        ),
      },
    ];

    return (
      <div className="flex w-full items-center justify-center">
        <Select className="w-[300px]">
          <Select.ListBox>
            {options.map(({ name, item }) => (
              <Select.Item key={name} id={name} name={name} item={item} />
            ))}
          </Select.ListBox>
        </Select>
      </div>
    );
  },
};

const SearchablesSelect = ({ selectOptions, icon, ...rest }: StoryArgs) => {
  const { contains } = useFilter({ sensitivity: "base" });

  return (
    <div className="flex w-full items-center justify-center">
      <Select {...rest}>
        <Select.SearchableListBox filter={contains}>
          {selectOptions.map((name) => (
            <Select.Item key={name} id={name} name={name} icon={icon} />
          ))}
        </Select.SearchableListBox>
      </Select>
    </div>
  );
};

export const Searchable: Story = {
  render: SearchablesSelect,
};

export const WithSections: Story = {
  render: ({ selectOptions, icon, ...rest }) => (
    <div className="flex w-full items-center justify-center">
      <Select {...rest}>
        <Select.ListBox>
          <Select.Item id="all" name="All animals" />
          <Select.Section title="Animals">
            {selectOptions.map((name) => (
              <Select.Item key={name} id={name} name={name} icon={icon} />
            ))}
          </Select.Section>
        </Select.ListBox>
      </Select>
    </div>
  ),
};

export const Pending: Story = {
  args: {
    isPending: true,
    selectOptions: [],
  },
};

export const Terminal: Story = {
  args: {
    label: undefined,
    supportText: undefined,
    variant: "terminal",
  },
};

export const Nav: Story = {
  args: {
    label: undefined,
    supportText: undefined,
    variant: "nav",
    value: "Kitten",
    icon: "diagram-project",
  },
};

export const Heading: Story = {
  args: {
    label: undefined,
    supportText: undefined,
    variant: "heading",
    value: "Kitten",
    triggerDescription: "A juvenile domestic cat",
    popoverFooter: "Pick an animal to see its details.",
    className: undefined,
  },
};
