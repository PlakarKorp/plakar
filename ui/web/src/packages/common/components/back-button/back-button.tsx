import { Button } from "@plakar-ui/ui/components";
import { LinkProps } from "react-aria-components";

type BackButtonProps = Pick<LinkProps, "href">;

export function BackButton({ href }: BackButtonProps) {
  const handleHistoryBack = () => (!href ? window.history.back() : undefined);

  return (
    <Button
      leftIcon="arrow-left"
      variant="tertiary"
      className="w-fit"
      href={href}
      onPress={handleHistoryBack}
      size="sm"
    >
      Back
    </Button>
  );
}
