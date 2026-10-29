import { Drawer } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";

import { AssistantChat } from "./assistant-chat";

import { getAssistantStatusQueryOptions } from "~/api/assistant/queries";
import { useAssistantChat } from "~/hooks/use-assistant-chat";

type AssistantDrawerProps = {
  open: boolean;
  setOpen: (open: boolean) => void;
};

export function AssistantDrawer({ open, setOpen }: AssistantDrawerProps) {
  // The conversation lives here, outside the drawer, so closing the drawer
  // does not lose it.
  const chat = useAssistantChat();
  const { data: status } = useQuery(getAssistantStatusQueryOptions());

  return (
    <Drawer open={open} setOpen={setOpen} title="Assistant">
      <AssistantChat chat={chat} status={status} />
    </Drawer>
  );
}
