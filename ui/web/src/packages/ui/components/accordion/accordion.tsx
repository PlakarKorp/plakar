import clsx from "clsx";
import {
  Button,
  Disclosure,
  DisclosureGroup,
  DisclosureGroupProps,
  DisclosurePanel,
  DisclosureProps,
  Heading,
} from "react-aria-components";

import { Typography, Icon, IconType } from "..";

type AccordionRootProps = DisclosureGroupProps;

function AccordionRoot({ children, className, ...props }: AccordionRootProps) {
  return (
    <DisclosureGroup
      className={clsx("flex flex-col gap-4", className)}
      {...props}
    >
      {children}
    </DisclosureGroup>
  );
}

type AccordionItemProps = DisclosureProps & {
  title: string;
  truncateTitle?: boolean;
  children?: React.ReactNode;
  icon?: IconType;
};

function AccordionItem({
  title,
  truncateTitle = true,
  children,
  icon,
  className,
  ...props
}: AccordionItemProps) {
  return (
    <Disclosure className={clsx("group/accordion", className)} {...props}>
      <Heading>
        <Button
          slot="trigger"
          className={clsx(
            "flex w-full items-center justify-between gap-2",
            !children ? "text-disable" : "text-primary cursor-pointer"
          )}
          isDisabled={!children}
        >
          <span className="flex items-center gap-3">
            {icon && <Icon type={icon} size="md" />}
            <Typography
              as="span"
              size="sm"
              weight="medium"
              className={clsx("text-left", truncateTitle && "truncate")}
            >
              {title}
            </Typography>
          </span>
          <Icon
            type="chevron-right"
            aria-hidden
            size="sm"
            className="transition-transform duration-200 group-data-[expanded]/accordion:rotate-90"
          />
        </Button>
      </Heading>
      <DisclosurePanel className="group-data-[expanded]/accordion:mt-2">
        {children}
      </DisclosurePanel>
    </Disclosure>
  );
}

export const Accordion = Object.assign(AccordionRoot, {
  Item: AccordionItem,
});
