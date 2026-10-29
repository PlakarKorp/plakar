import { Disclosure, Spinner, Tag, Typography } from "@plakar-ui/ui/components";

import { ToolCall as ToolCallType } from "~/hooks/use-assistant-chat";

type ToolCallProps = {
  call: ToolCallType;
};

export function ToolCall({ call }: ToolCallProps) {
  return (
    <div className="border-primary bg-secondary flex flex-col rounded-lg border px-1 py-1">
      <div className="flex items-center gap-2">
        <Disclosure title={call.name} className="min-w-0 flex-1">
          <div className="flex flex-col gap-2 px-2 pb-1">
            {call.input !== undefined && (
              <ToolCallText label="Input" text={call.input} />
            )}
            {call.output !== undefined && (
              <ToolCallText label="Output" text={call.output} />
            )}
            {call.truncated && (
              <Typography size="xs" className="text-tertiary">
                The output was cut short before reaching the model.
              </Typography>
            )}
          </div>
        </Disclosure>
        {call.status === "running" ? (
          <Spinner size="sm" className="h-auto pr-2" />
        ) : (
          call.isError && (
            <Tag variant="red" size="xs" className="mr-2">
              Failed
            </Tag>
          )
        )}
      </div>
    </div>
  );
}

type ToolCallTextProps = {
  label: string;
  text: string;
};

function ToolCallText({ label, text }: ToolCallTextProps) {
  return (
    <div className="flex flex-col gap-0.5">
      <Typography size="xs" weight="medium" className="text-tertiary">
        {label}
      </Typography>
      <Typography
        as="pre"
        size="xs"
        monospace
        className="bg-primary text-secondary max-h-64 overflow-auto rounded-md p-2 break-all whitespace-pre-wrap"
      >
        {text}
      </Typography>
    </div>
  );
}
