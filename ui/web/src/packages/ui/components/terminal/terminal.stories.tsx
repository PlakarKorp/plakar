import type { Meta, StoryObj } from "@storybook/react-vite";

import { Terminal } from "./terminal";

const meta = {
  title: "design-system/Terminal",
  component: Terminal,
  tags: ["autodocs"],
} satisfies Meta<typeof Terminal>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <Terminal>
      <Terminal.Command
        prompt={
          <Terminal.Prompt username="plakar" host="plakar.dev" path="~" />
        }
        command="plakar at /tmp/toto create -no-encryption"
      />
      <Terminal.Command
        prompt={
          <Terminal.Prompt username="plakar" host="plakar.dev" path="~" />
        }
        command="plakar at /tmp/toto backup /etc"
      />
      <Terminal.Command
        prompt={
          <Terminal.Prompt username="plakar" host="plakar.dev" path="~" />
        }
        command="plakar at /tmp/toto ls"
      >
        <Terminal.CommandOutputLine>
          2025-04-04T23:22:53+02:00 warn: failed to connect to agent, falling
          back to -no-agent
        </Terminal.CommandOutputLine>
        <Terminal.CommandOutputLine>
          2025-04-04T09:19:04Z 8ea7d4c8 3.1 MB 0s /private/etc
        </Terminal.CommandOutputLine>
        <Terminal.CommandOutputLine>
          2025-04-03T14:34:03Z 57586d66 3.1 MB 0s /
        </Terminal.CommandOutputLine>
        <Terminal.CommandOutputLine>
          2025-04-02T12:32:32Z 9c6cd2c7 443 MB 6s
          /Users/niluje/dev/plakar/plakar-ui
        </Terminal.CommandOutputLine>
        <Terminal.CommandOutputLine>
          2025-03-31T20:24:22Z acb4de80 6.6 GB 12s /Users/niluje/Movies/How I
          Met Your Mother (2005) Season 1-9 S01-S09 (1080p MIXED x265 HEVC 10bit
          AAC 5.1 Silence)/Season 7 (WEB)
        </Terminal.CommandOutputLine>
        <Terminal.CommandOutputLine>
          2025-03-31T20:24:22Z acb4de80 6.6 GB 12s /Users/niluje/Movies/How I
          Met Your Mother (2005) Season 1-9 S01-S09 (1080p MIXED x265 HEVC 10bit
          AAC 5.1 Silence)/Season 7 (WEB)
        </Terminal.CommandOutputLine>
        <Terminal.CommandOutputLine>
          2025-03-27T10:12:58Z 29f6cfe1 26 MB 0s /Users/niluje/Desktop/CAP
          cuisine
        </Terminal.CommandOutputLine>
        <Terminal.CommandOutputLine>
          2025-03-27T10:07:59Z e6b24c32 338 MB 1s
          /Users/niluje/dev/plakar/plakar
        </Terminal.CommandOutputLine>
      </Terminal.Command>
    </Terminal>
  ),
};
