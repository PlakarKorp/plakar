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
    <div className="relative flex flex-col-reverse items-start gap-6 xl:grid xl:grid-cols-[minmax(0,1fr)_20rem]">
      <article className="border-secondary bg-primary w-full overflow-auto rounded-xl border p-6 md:p-8">
        <Markdown content={documentation} />
      </article>
      <div className="flex w-full flex-col gap-4 xl:sticky xl:top-4">
        {aside}
      </div>
    </div>
  );
}
