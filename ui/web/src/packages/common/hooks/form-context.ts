import { createFormHookContexts } from "@tanstack/react-form";

const {
  fieldContext,
  formContext,
  useFieldContext,
  useFormContext: useUntypedFormContext,
} = createFormHookContexts();

export { fieldContext, formContext, useFieldContext };

// `useFormContext()` doesn't propagate the form data type (TS can't infer it),
// so we centralize the cast here. See TanStack Form discussion:
// https://github.com/TanStack/form/discussions/1335
export function useFormContext(): ReturnType<typeof useUntypedFormContext>;
export function useFormContext<T>(): T;
export function useFormContext<T>(): T {
  return useUntypedFormContext() as unknown as T;
}
