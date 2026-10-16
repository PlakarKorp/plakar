import { Button } from "@plakar-ui/ui/components";
import { LinkProps } from "react-aria-components";

import { Error } from "./error";

type NotFoundErrorProps = Pick<LinkProps, "href">;

export function NotFoundError({ href }: NotFoundErrorProps) {
  return (
    <Error
      illustration="juggles_nuts"
      title="404 - Page not found"
      content={<>Sorry, we couldn't find the page you're looking for.</>}
      action={
        <Button href={href} variant="secondary">
          Return to Dashboard
        </Button>
      }
    />
  );
}
