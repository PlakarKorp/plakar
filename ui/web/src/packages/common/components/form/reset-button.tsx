import { Button, ButtonProps } from "@plakar-ui/ui/components";
import { PressEvent } from "react-aria-components";

import { useFormContext } from "../../hooks/form-context";

export function ResetButton({
  children,
  isDisabled,
  onPress,
  ...props
}: Omit<ButtonProps, "type" | "variant">) {
  const form = useFormContext();

  const handleReset = (e: PressEvent) => {
    form.reset();
    onPress?.(e);
  };

  return (
    <form.Subscribe selector={(s) => [s.isDirty]}>
      {([isDirty]) => (
        <Button
          type="reset"
          isDisabled={!isDirty || isDisabled}
          onPress={handleReset}
          variant="secondary"
          {...props}
        >
          {children}
        </Button>
      )}
    </form.Subscribe>
  );
}
