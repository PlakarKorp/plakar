import clsx from "clsx";
import {
  Button,
  Disclosure as RADisclosure,
  DisclosurePanel,
  DisclosureProps,
  Heading,
} from "react-aria-components";

import { Icon, Typography } from "..";

type DisclosureComponentProps = DisclosureProps & {
  title: string;
  children: React.ReactNode;
};

export function Disclosure({
  title,
  children,
  className,
  ...props
}: DisclosureComponentProps) {
  return (
    <RADisclosure className={clsx("group/disclosure", className)} {...props}>
      <Heading>
        <Button
          slot="trigger"
          className="text-tertiary data-hovered:text-secondary flex cursor-pointer items-center gap-1 rounded-md px-2 py-0.5"
        >
          <Icon
            type="chevron-right"
            aria-hidden
            size="sm"
            className="transition-transform duration-200 group-data-[expanded]/disclosure:rotate-90"
          />
          <Typography as="span" size="sm" weight="medium">
            {title}
          </Typography>
        </Button>
      </Heading>
      <DisclosurePanel className="group-data-[expanded]/disclosure:mt-1">
        {children}
      </DisclosurePanel>
    </RADisclosure>
  );
}
