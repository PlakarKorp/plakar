import type { Meta, StoryObj } from "@storybook/react-vite";

import { Typography } from "./typography/typography";

type ColorSwatchProps = {
  token: string;
};

const GROUP_DEFINITIONS: { name: string; match: (token: string) => boolean }[] =
  [
    {
      name: "Background",
      match: (t) =>
        t.startsWith("--background-background") ||
        t.startsWith("--background-color-"),
    },
    { name: "Text", match: (t) => t.startsWith("--text-color-") },
    { name: "Borders", match: (t) => t.startsWith("--border-color-") },
    { name: "Outlines", match: (t) => t.startsWith("--outline-color-") },
  ];

function getAllCssVariables(): string[] {
  if (typeof window === "undefined") return [];

  const style = getComputedStyle(document.documentElement);
  const tokens: string[] = [];

  for (let i = 0; i < style.length; i++) {
    const prop = style.item(i);
    if (prop.startsWith("--")) {
      tokens.push(prop);
    }
  }

  // remove duplicates and sort
  return Array.from(new Set(tokens)).sort();
}

function buildColorGroups(): Record<string, string[]> {
  const tokens = getAllCssVariables();

  const groups: Record<string, string[]> = {};
  const used = new Set<string>();

  // Assign tokens to configured groups
  for (const { name, match } of GROUP_DEFINITIONS) {
    const groupTokens = tokens.filter((t) => match(t));
    if (groupTokens.length > 0) {
      groups[name] = groupTokens;
      groupTokens.forEach((t) => used.add(t));
    }
  }

  return groups;
}

const ColorSwatch = ({ token }: ColorSwatchProps) => {
  return (
    <div className="border-primary bg-primary flex items-center gap-3 rounded-lg border p-3">
      <div
        className="border-primary h-10 w-10 shrink-0 rounded-md border"
        style={{
          backgroundColor: `var(${token})`,
        }}
      />
      <Typography size="xs" monospace>
        {token
          .replace("--background-color", "bg")
          .replace("--background", "bg")
          .replace("--text-color", "text")
          .replace("--border-color", "border")
          .replace("--outline-color", "outline")}
      </Typography>
    </div>
  );
};

const ColorGroup = ({ title, tokens }: { title: string; tokens: string[] }) => (
  <section className="space-y-3">
    <Typography as="h2" variant="heading" size="sm" weight="medium">
      {title}
    </Typography>
    <div className="grid gap-3 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4">
      {tokens.map((token) => (
        <ColorSwatch key={token} token={token} />
      ))}
    </div>
  </section>
);

type ColorsDisplayProps = {
  title: string;
  description: string;
  filterGroup?: string;
};

const ColorsDisplay = ({
  title,
  description,
  filterGroup,
}: ColorsDisplayProps) => {
  const colorGroups = buildColorGroups();

  const displayGroups = filterGroup
    ? { [filterGroup]: colorGroups[filterGroup] || [] }
    : colorGroups;

  return (
    <div className="text-primary">
      <div className="mx-auto flex max-w-6xl flex-col gap-8">
        <header>
          <Typography variant="heading" weight="semibold">
            {title}
          </Typography>
          <Typography size="sm" className="text-secondary">
            {description}
          </Typography>
        </header>

        <div className="flex flex-col gap-8">
          {Object.entries(displayGroups).map(([group, tokens]) => (
            <ColorGroup key={group} title={group} tokens={tokens} />
          ))}
        </div>
      </div>
    </div>
  );
};

const meta: Meta = {
  title: "Theme/Colors",
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;

type Story = StoryObj<typeof meta>;

export const AllColors: Story = {
  render: () => (
    <ColorsDisplay
      title="Color classes"
      description="All CSS custom properties defined in the theme"
    />
  ),
};

export const Background: Story = {
  render: () => (
    <ColorsDisplay
      title="Background Colors"
      description="Background color classes in the theme"
      filterGroup="Background"
    />
  ),
};

export const Text: Story = {
  render: () => (
    <ColorsDisplay
      title="Text Colors"
      description="Text color classes in the theme"
      filterGroup="Text"
    />
  ),
};

export const Borders: Story = {
  render: () => (
    <ColorsDisplay
      title="Border Colors"
      description="Border color classes in the theme"
      filterGroup="Borders"
    />
  ),
};

export const Outlines: Story = {
  render: () => (
    <ColorsDisplay
      title="Outline Colors"
      description="Outline color classes in the theme"
      filterGroup="Outlines"
    />
  ),
};
