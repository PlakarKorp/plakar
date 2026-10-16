import { Meta, StoryObj } from "@storybook/react-vite";

import { Illustration } from "./illustration";
import { IllustrationTypes } from "./illustration.constants";

function Template() {
  return (
    <table className="w-full">
      <thead>
        <tr>
          <th className="border-tertiary border p-2 text-center">
            Illustration
          </th>
          <th className="border-tertiary border p-2">Type</th>
        </tr>
      </thead>
      <tbody>
        {[...IllustrationTypes].map((type) => (
          <tr key={type}>
            <td className="border-tertiary border p-2 text-center">
              <Illustration type={type} width={100} />
            </td>
            <td className="border-tertiary border p-2">{type}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

const meta = {
  title: "design-system/Illustration",
  tags: ["autodocs"],
} satisfies Meta<typeof Template>;

export default meta;
type Story = StoryObj<typeof meta>;

export const AllIllustrations: Story = {
  render: () => <Template />,
};
