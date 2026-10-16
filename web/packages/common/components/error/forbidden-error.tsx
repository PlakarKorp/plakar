import { Button } from "@plakar-ui/ui/components";
import { LinkProps } from "react-aria-components";

import { Error } from "./error";

type ForbiddenErrorProps = Pick<LinkProps, "href">;

export function ForbiddenError({ href }: ForbiddenErrorProps) {
  return (
    <Error
      illustration="shrug"
      icon="lock"
      title="403 - Not allowed"
      content={
        <>
          Your permissions do not reach this page. Someone who administers your
          organization can grant them.
        </>
      }
      action={
        <Button href={href} variant="secondary">
          Return to Dashboard
        </Button>
      }
    />
  );
}
