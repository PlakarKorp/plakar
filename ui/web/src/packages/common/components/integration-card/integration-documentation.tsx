import { Markdown } from "@plakar-ui/ui/components";
import { ReactNode } from "react";

type IntegrationDocumentationProps = {
  documentation: string;
  aside: ReactNode;
};

export function IntegrationDocumentation({
  documentation,
  aside,
}: IntegrationDocumentationProps) {
  return (
    <div className="relative flex flex-col-reverse items-start gap-4 xl:grid xl:grid-cols-[auto_26rem]">
      <div className="w-full overflow-auto">
        <Markdown content={documentation} />
      </div>
      <div className="flex w-full flex-col gap-4 xl:sticky xl:top-4">
        {aside}
      </div>
    </div>
  );
}
