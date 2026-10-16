import { ComponentProps } from "react";

import { AsyncSelectField } from "./async-select-field";
import { TextAreaField } from "./text-area-field";
import { TextField } from "./text-field";

export type TextFieldProps = ComponentProps<typeof TextField>;
export type TextAreaProps = ComponentProps<typeof TextAreaField>;
export type AsyncSelectFieldList = ComponentProps<
  typeof AsyncSelectField
>["list"];
