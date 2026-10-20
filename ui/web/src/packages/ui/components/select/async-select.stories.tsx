import type { Meta, StoryObj } from "@storybook/react-vite";
import { useAsyncList } from "react-stately";

import { AsyncSelect } from "./async-select";
import { SelectOption } from "./select";

const PER_PAGE = 20;
const TOTAL = 95;

function AsyncSelectStory({ search }: { search: boolean }) {
  const list = useAsyncList<SelectOption, number>({
    async load({ cursor, filterText }) {
      const page = cursor ?? 0;
      await new Promise((resolve) => setTimeout(resolve, 500));
      const all = Array.from({ length: TOTAL }, (_, i) => ({
        id: `item-${i + 1}`,
        name: `Item ${i + 1}`,
      })).filter((item) =>
        item.name.toLowerCase().includes((filterText ?? "").toLowerCase())
      );
      return {
        items: all.slice(page * PER_PAGE, (page + 1) * PER_PAGE),
        cursor: all.length > (page + 1) * PER_PAGE ? page + 1 : undefined,
      };
    },
  });
  return (
    <div className="flex w-full items-center justify-center">
      <AsyncSelect
        label="Label"
        placeholder="Select an item"
        className="w-[240px]"
        list={list}
        search={search}
      />
    </div>
  );
}

const meta: Meta<typeof AsyncSelectStory> = {
  title: "design-system/Async Select",
  component: AsyncSelectStory,
  tags: ["autodocs"],
  args: { search: false },
};

export default meta;

type Story = StoryObj<typeof AsyncSelectStory>;

export const Default: Story = {};

export const Searchable: Story = {
  args: { search: true },
};
