import { IconType } from "@plakar-ui/ui/components";

type MimeFilters = {
  id: string;
  icon: IconType;
  name: string;
  contentTypes: string[];
};

export const MIME_FILTERS: MimeFilters[] = [
  {
    id: "images",
    icon: "md-image",
    name: "Images",
    contentTypes: ["image"],
  },
  {
    id: "pdfs",
    icon: "md-picture-as-pdf",
    name: "PDFs",
    contentTypes: ["application/pdf"],
  },
  {
    id: "videos",
    icon: "md-video-file",
    name: "Videos",
    contentTypes: ["video"],
  },
  {
    id: "audio",
    icon: "md-audio-file",
    name: "Audio",
    contentTypes: ["audio"],
  },
  {
    id: "archives",
    icon: "md-folder-zip",
    name: "Archives",
    contentTypes: [
      "application/zip", // .zip
      "application/x-tar", // .tar
      "application/gzip", // .gz, .tgz, .tar.gz
      "application/x-bzip2", // .bz2, .tar.bz2
      "application/x-xz", // .xz, .tar.xz
      "application/x-7z-compressed", // .7z
      "application/x-lzma", // .lzma, .tar.lzma
      "application/vnd.rar", // .rar
      "application/x-compressed-tar", // .tar.gz (.tgz)
      "application/x-bzip-compressed-tar", // .tar.bz2 (.tbz)
      "application/x-zstd", // .zst, .zstd (unofficial but used)
      "application/x-lzip", // .lz, .tar.lz
    ],
  },
  {
    id: "text",
    icon: "md-file-copy",
    name: "Text files",
    contentTypes: ["text"],
  },
];
