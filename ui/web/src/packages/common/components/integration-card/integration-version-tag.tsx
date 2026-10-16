import { Tag, TagVariant } from "@plakar-ui/ui/components";

type IntegrationStageTagProps = {
  stage: string;
};

export function IntegrationStageTag({ stage }: IntegrationStageTagProps) {
  if (!stage) {
    return null;
  }

  const colors: Record<string, (typeof TagVariant)[number]> = {
    stable: "green",
    testing: "blue",
    beta: "yellow",
    devel: "red",
    alpha: "red",
  };

  const found = Object.keys(colors).find(([key]) => {
    return stage.toLowerCase().includes(key);
  });

  const variant = found ? colors[found] : "red";
  const display = found ? found : stage;

  return (
    <Tag variant={variant} size="sm">
      {display}
    </Tag>
  );
}
