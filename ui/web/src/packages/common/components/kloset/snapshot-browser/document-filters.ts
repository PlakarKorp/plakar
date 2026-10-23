import { IconType } from "@plakar-ui/ui/components";

export type DocumentFilter = {
  id: string;
  icon: IconType;
  name: string;
  noun: string;
  contentTypes: string[];
};

export const DOCUMENT_FILTERS: DocumentFilter[] = [
  {
    id: "pdfs",
    icon: "md-picture-as-pdf",
    name: "PDFs",
    noun: "PDFs",
    contentTypes: ["application/pdf"],
  },
  {
    id: "text",
    icon: "file-lines",
    name: "Text",
    noun: "text files",
    contentTypes: ["text"],
  },
  {
    id: "data",
    icon: "database",
    name: "Data",
    noun: "data files",
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

/**
 * Tells whether a content type belongs to a format, as the search endpoint
 * only compares the part before the slash when it does not recurse.
 */
export function matchesDocumentFilter(
  filter: DocumentFilter,
  contentType: string
) {
  const [type] = contentType.split(";");
  return filter.contentTypes.some(
    (match) => type === match || type.startsWith(`${match}/`)
  );
}
