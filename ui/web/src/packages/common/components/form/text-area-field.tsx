import {
  FileImportButton,
  TextArea,
  TextInputProps,
} from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";
import { addToast } from "../toast";

type TextAreaFieldProps = Omit<
  TextInputProps,
  "value" | "onChange" | "onBlur" | "isInvalid"
> & {
  withFileImport?: boolean;
  acceptedFileTypes?: string[];
  fileImportLabel?: string;
  rows?: number;
};

export function TextAreaField({
  supportText,
  withFileImport = false,
  acceptedFileTypes,
  fileImportLabel = "Select file",
  ...props
}: TextAreaFieldProps) {
  const field = useFieldContext<string | string[]>();
  const isReadOnly = useFormReadOnly() || props.isReadOnly;

  const { value, meta } = field.state;
  const { errors } = meta;

  const stringValue = Array.isArray(value) ? value.join("\n") : (value ?? "");

  const handleChange = (newValue: string) => {
    if (Array.isArray(field.state.value)) {
      field.handleChange(newValue.split(/\r?\n/));
    } else {
      field.handleChange(newValue);
    }
  };

  const handleFileSelect = async (file: File) => {
    try {
      const text = await file.text();
      handleChange(text);
    } catch {
      addToast({
        content: {
          children: `Could not read ${file.name}`,
          variant: "error",
        },
      });
    }
  };

  return (
    <TextArea
      value={stringValue}
      onChange={handleChange}
      onBlur={field.handleBlur}
      isInvalid={errors.length > 0}
      supportText={errors[0] || supportText}
      suffix={
        withFileImport && !isReadOnly ? (
          <FileImportButton
            onSelect={handleFileSelect}
            acceptedFileTypes={acceptedFileTypes}
          >
            {fileImportLabel}
          </FileImportButton>
        ) : undefined
      }
      {...props}
      isReadOnly={isReadOnly}
    />
  );
}
