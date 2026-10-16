import { Icon, IconSize } from "@plakar-ui/ui/components";

type FileIconProps = {
  contentType?: string;
  size?: IconSize;
};

export function FileIcon({ contentType, size = "lg" }: FileIconProps) {
  if (contentType?.startsWith("image")) {
    return <Icon type="md-image" size={size} />;
  } else if (contentType === "application/pdf") {
    return <Icon type="md-picture-as-pdf" size={size} />;
  } else if (contentType?.startsWith("video")) {
    return <Icon type="md-video-file" size={size} />;
  } else if (contentType?.startsWith("audio")) {
    return <Icon type="md-audio-file" size={size} />;
  } else if (contentType === "application/zip") {
    return <Icon type="md-folder-zip" size={size} />;
  }
  return <Icon type="md-file-copy" size={size} />;
}
