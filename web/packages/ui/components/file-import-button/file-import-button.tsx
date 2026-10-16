import { ReactNode } from "react";
import { FileTrigger } from "react-aria-components";

import { Button } from "../button/button";
import { ButtonSize, ButtonVariant } from "../button/button.constants";

type FileImportButtonProps = {
  acceptedFileTypes?: string[];
  onSelect: (file: File) => void;
  variant?: ButtonVariant;
  size?: ButtonSize;
  children?: ReactNode;
};

export function FileImportButton({
  acceptedFileTypes,
  onSelect,
  variant = "tertiary",
  size = "sm",
  children = "Select file",
}: FileImportButtonProps) {
  return (
    <FileTrigger
      acceptedFileTypes={acceptedFileTypes}
      onSelect={(files) => {
        const file = files?.item(0);
        if (file) {
          onSelect(file);
        }
      }}
    >
      <Button
        leftIcon="file-import"
        type="button"
        variant={variant}
        size={size}
      >
        {children}
      </Button>
    </FileTrigger>
  );
}
