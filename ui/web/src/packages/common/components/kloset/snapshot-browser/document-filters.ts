import { IconType } from "@plakar-ui/ui/components";

export type DocumentFilter = {
  id: string;
  icon: IconType;
  name: string;
  contentTypes: string[];
};

export const DOCUMENT_FILTERS: DocumentFilter[] = [
  {
    id: "pdfs",
    icon: "md-picture-as-pdf",
    name: "PDFs",
    contentTypes: ["application/pdf"],
  },
  {
    id: "text",
    icon: "file-lines",
    name: "Text",
    contentTypes: ["text"],
  },
  {
    id: "data",
    icon: "database",
    name: "Data",
    contentTypes: [
      "application/json",
      "application/xml",
      "application/x-yaml",
      "application/yaml",
      "application/toml",
      "application/javascript",
    ],
  },
];
