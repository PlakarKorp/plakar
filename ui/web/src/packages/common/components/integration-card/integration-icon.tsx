import { Icon, IconSize } from "@plakar-ui/ui/components";
import { useState } from "react";

import { getIntegrationIcon, hasIntegrationIcon } from "../../api/enrich";

type IntegrationIconProps = {
  name: string;
  icon?: string;
  size?: IconSize;
};

export function IntegrationIcon({
  name,
  icon,
  size = "2xl",
}: IntegrationIconProps) {
  const [failedIcon, setFailedIcon] = useState<string>();

  if (!hasIntegrationIcon(name) && icon && icon !== failedIcon) {
    return (
      <img
        src={icon}
        className="h-full w-full object-cover object-center"
        alt=""
        onError={() => setFailedIcon(icon)}
      />
    );
  }

  return <Icon type={getIntegrationIcon(name)} size={size} />;
}
