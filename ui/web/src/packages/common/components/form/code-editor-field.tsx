import {
  CodeEditor,
  CodeEditorProps,
  FileImportButton,
} from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";
import { addToast } from "../toast";

type CodeEditorFieldProps = Omit<
  CodeEditorProps,
  "value" | "onChange" | "onBlur" | "isInvalid"
> & {
  withFileImport?: boolean;
  acceptedFileTypes?: string[];
};

export function CodeEditorField({
  supportText,
  withFileImport = false,
  acceptedFileTypes,
  ...props
}: CodeEditorFieldProps) {
  const field = useFieldContext<string>();
  const isReadOnly = useFormReadOnly();

  const { value, meta } = field.state;
  const { errors } = meta;

  const handleFileSelect = async (file: File) => {
    try {
      field.handleChange(await file.text());
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
    <CodeEditor
      value={value ?? ""}
      onChange={field.handleChange}
      onBlur={field.handleBlur}
      isInvalid={errors.length > 0}
      supportText={errors[0] || supportText}
      suffix={
        withFileImport && !isReadOnly ? (
          <FileImportButton
            onSelect={handleFileSelect}
            acceptedFileTypes={acceptedFileTypes}
          />
        ) : undefined
      }
      {...props}
      editable={isReadOnly ? false : props.editable}
    />
  );
}
