import {
  Icon,
  IconSize,
  IconType,
  SelectOption,
  Typography,
} from "@plakar-ui/ui/components";

import { StorageTier, StorageTiers } from "../../api/storage-tier";

export const STORAGE_TIER_DISPLAY: Record<
  StorageTier,
  { label: string; icon: IconType; iconColor: string; description: string }
> = {
  hot: {
    label: "Standard",
    icon: "box-archive",
    iconColor: "text-secondary",
    description: "Storage available at all times",
  },
  cold: {
    label: "Cold",
    icon: "snowflake",
    iconColor: "text-sky-500",
    description:
      "Archival storage that must be retrieved before access, such as Glacier",
  },
};

export const getStorageTierOptions = (
  showDescription: boolean
): SelectOption[] => {
  return StorageTiers.map((tier) => {
    const { label, icon, iconColor, description } = STORAGE_TIER_DISPLAY[tier];

    return {
      id: tier,
      name: label,
      item: (
        <div className="flex items-center gap-2">
          <Icon type={icon} size="md" className={iconColor} />

          <div className="flex items-baseline gap-1">
            <Typography
              as="span"
              variant="body"
              size="sm"
              className="text-secondary"
            >
              {label}
            </Typography>
            {showDescription && (
              <Typography
                as="span"
                variant="body"
                size="xs"
                className="text-tertiary"
              >
                - {description}
              </Typography>
            )}
          </div>
        </div>
      ),
    };
  });
};

export const getStorageTierIcon = (
  temperature: StorageTier | null | undefined,
  size: IconSize = "md"
) => {
  if (!temperature || temperature === "hot") {
    return null;
  }

  const { icon, iconColor, label } = STORAGE_TIER_DISPLAY[temperature];

  return (
    <Icon
      type={icon}
      size={size}
      role="img"
      aria-label={`${label} store`}
      title={`${label} store`}
      className={iconColor}
    />
  );
};
