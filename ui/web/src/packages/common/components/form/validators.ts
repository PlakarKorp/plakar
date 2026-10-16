import { AnyFieldApi } from "@tanstack/react-form";

/** What a field is checked against, whatever moment it is checked at. */
type Check<TValue> = (args: { value: TValue }) => string | undefined;

/**
 * Drops what a blur or a submission said. A field stops being told off the
 * moment the reader works on it again, under either policy below.
 */
const forget = ({ fieldApi }: { fieldApi: AnyFieldApi }) => {
  fieldApi.setMeta((prev) => ({
    ...prev,
    errorMap: { ...prev.errorMap, onBlur: undefined, onSubmit: undefined },
  }));
  return undefined;
};

/** Reports the check at every opportunity, from the first keystroke on. */
export function eagerly<TValue>(check: Check<TValue>) {
  return {
    onChange: (args: { value: TValue; fieldApi: AnyFieldApi }) => {
      forget(args);
      return check(args);
    },
    onBlur: check,
    onSubmit: check,
  };
}

/**
 * Reports the check only once the form is submitted. For a field emptied on the
 * way to being refilled, or given up by the form itself when the choice stops
 * applying: complaining in between would be complaining about a step.
 */
function onSubmitOnly<TValue>(check: Check<TValue>) {
  return {
    onChange: forget,
    onSubmit: check,
  };
}

const checkEmpty = ({ value }: { value: string | null | undefined }) => {
  if (!value) {
    return "This field is required.";
  }
  return undefined;
};

export const requiredValidators = eagerly(checkEmpty);

export const requiredOnSubmitValidators = onSubmitOnly(checkEmpty);

const EMAIL_PATTERN = /^[^\s@]+@[^\s@.]+(?:\.[^\s@.]+)+$/;

const checkEmail = ({ value }: { value: string | null | undefined }) => {
  const empty = checkEmpty({ value });
  if (empty) {
    return empty;
  }
  if (!EMAIL_PATTERN.test(value ?? "")) {
    return "Please enter a valid email address.";
  }
  return undefined;
};

/**
 * Required like {@link requiredValidators}, and checked for shape once the
 * reader is done with the field: an address is malformed for as long as it is
 * half-typed, so only emptiness is reported while typing.
 */
export const requiredEmailValidators = {
  ...eagerly(checkEmpty),
  onBlur: checkEmail,
  onSubmit: checkEmail,
};

const checkEmptyList = ({ value }: { value?: readonly string[] }) =>
  value?.length ? undefined : "Select at least one item.";

/**
 * The list counterpart of {@link requiredOnSubmitValidators}: it asks for at
 * least one item, and only says so once the form is submitted.
 */
export const requiredListOnSubmitValidators = onSubmitOnly(checkEmptyList);
