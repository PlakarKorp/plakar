import { createContext, useContext } from "react";

export const FormReadOnlyContext = createContext(false);

/**
 * Whether the surrounding form is read-only.
 *
 * Field components fold this into what they render, so a form marked read-only
 * once through `FormElement` reaches fields it does not know about -- including
 * the ones added after the gate was written. Components that reach for a UI
 * primitive directly, or that render an edit affordance of their own, read it
 * here.
 *
 * Inputs with a read-only rendering take it as `isReadOnly`. Checkboxes,
 * toggles, radios, multi-selects and tag toggles have none, so they take it as
 * `isDisabled` instead.
 */
export function useFormReadOnly() {
  return useContext(FormReadOnlyContext);
}
