import { ComponentProps } from "react";
import { Form, FormContext } from "react-aria-components";

import { FormReadOnlyContext } from "../../hooks/use-form-read-only";

type FormElementProps = {
  handleSubmit: () => void;
  isReadOnly?: boolean;
} & Omit<ComponentProps<typeof Form>, "onSubmit" | "validationBehavior">;

export function FormElement({
  handleSubmit,
  isReadOnly = false,
  children,
  ...props
}: FormElementProps) {
  return (
    // Reset the context so a form nested inside another form doesn't inherit
    // the parent form's context.
    <FormContext.Provider value={null}>
      <Form
        {...props}
        validationBehavior="aria"
        onSubmit={(e) => {
          e.preventDefault();
          e.stopPropagation();
          // Enter in a field submits without going through a button, so the
          // read-only form has to refuse here rather than rely on its controls.
          if (isReadOnly) return;
          handleSubmit();
        }}
      >
        <FormReadOnlyContext.Provider value={isReadOnly}>
          {children}
        </FormReadOnlyContext.Provider>
      </Form>
    </FormContext.Provider>
  );
}
