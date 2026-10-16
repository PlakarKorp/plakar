import { FloatingOverlayButton } from "@plakar-ui/common/components";
import { Button, Illustration, Typography } from "@plakar-ui/ui/components";

type FloatingSupportButtonProps = {
  bottom?: string;
};

export function FloatingSupportButton({ bottom }: FloatingSupportButtonProps) {
  return (
    <FloatingOverlayButton
      icon="circle-question"
      content={
        <div className="relative space-y-4">
          <Typography className="from-accent to-accent/80 text-on-brand bg-linear-to-r px-4 py-2">
            How can we help?
          </Typography>

          <div className="mr-20 ml-4 flex flex-col items-center gap-1.5">
            <Button
              leftIcon="github"
              variant="secondary"
              href="https://github.com/PlakarKorp/plakar/issues/new/choose"
              target="_blank"
              className="w-full"
            >
              Report an issue
            </Button>
            <div className="text-tertiary text-xs leading-none">
              Best for bugs and tracking
            </div>
          </div>

          <div className="mr-20 ml-4 flex flex-col items-center gap-1.5">
            <Button
              leftIcon="discord"
              variant="secondary"
              href="https://discord.gg/uuegtnF2Q5"
              target="_blank"
              className="w-full"
            >
              Ask on Discord
            </Button>
            <div className="text-tertiary text-xs leading-none">
              Best for quick help and chat
            </div>
          </div>

          <Illustration
            type="emperor"
            className="absolute -right-8 -bottom-10"
            width={120}
          />
        </div>
      }
      bottom={bottom}
    />
  );
}
