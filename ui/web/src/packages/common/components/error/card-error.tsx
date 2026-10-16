import { Button, Icon, Typography } from "@plakar-ui/ui/components";

import { APIResponseForbiddenError } from "../../api/errors";

type CardErrorProps = {
  error: Error | null;
  reset: () => void;
};

export function CardError({ error, reset }: CardErrorProps) {
  // Retrying a refusal only refuses again: a 403 is an answer, not a failure.
  if (error instanceof APIResponseForbiddenError) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 p-4 text-center">
        <Icon type="lock" size="lg" className="text-quaternary" />
        <Typography className="text-tertiary" size="sm">
          Your permissions do not reach this data.
        </Typography>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col items-center justify-center gap-2">
      <Typography className="text-secondary">Failed to load data.</Typography>
      <Button onPress={reset} variant="secondary">
        Retry
      </Button>
    </div>
  );
}
