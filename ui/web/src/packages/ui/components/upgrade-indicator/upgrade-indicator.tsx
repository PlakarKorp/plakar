import { Icon } from "..";

type UpgradeIndicatorProps = {
  label: string;
};

export function UpgradeIndicator({ label }: UpgradeIndicatorProps) {
  return (
    <div className="border-background-primary flex items-center justify-center rounded-full border-2">
      <Icon
        type="circle-up"
        size="sm"
        aria-hidden="true"
        className="text-accent"
      />
      <span className="sr-only">{label}</span>
    </div>
  );
}
