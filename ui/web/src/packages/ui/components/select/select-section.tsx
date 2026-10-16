import { ReactNode } from "react";
import { Header, ListBoxSection } from "react-aria-components";

import { Typography } from "../typography/typography";

type SelectSectionProps = {
  title: string;
  children: ReactNode;
};

export function SelectSection({ title, children }: SelectSectionProps) {
  return (
    <ListBoxSection className="border-secondary flex flex-col gap-1 not-first:mt-1 not-first:border-t not-first:pt-2">
      <Header className="px-2 pb-0.5">
        <Typography
          as="span"
          size="xs"
          weight="medium"
          className="text-tertiary"
        >
          {title}
        </Typography>
      </Header>
      {children}
    </ListBoxSection>
  );
}
