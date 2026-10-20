import { Card, Typography } from "@plakar-ui/ui/components";

export function StoppedScreen() {
  return (
    <div className="bg-secondary flex h-full items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <Card.HighlightedHeader
          illustration="waving"
          title="Plakar has quit"
          description="The process serving this page has exited."
        />
        <Card.Content padding="lg">
          <Typography size="sm" className="text-secondary">
            You can close this tab. To open the UI again, run{" "}
            <code>plakar ui</code> or start it the way you did before.
          </Typography>
        </Card.Content>
      </Card>
    </div>
  );
}
