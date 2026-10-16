import { Button, ButtonProps } from "@plakar-ui/ui/components";

import { useFormContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

export function SubmitButton({
  children,
  isDisabled,
  isPending,
  ...props
}: Omit<ButtonProps, "type" | "onPress">) {
  const form = useFormContext();
  const isReadOnly = useFormReadOnly();

  return (
    <form.Subscribe selector={(s) => [s.canSubmit, s.isSubmitting, s.isDirty]}>
      {([canSubmit, isSubmitting]) => (
        <Button
          type="submit"
          isDisabled={!canSubmit || isSubmitting || isReadOnly || isDisabled}
          isPending={isSubmitting || isPending}
          {...props}
        >
          {children}
        </Button>
      )}
    </form.Subscribe>
  );
}
