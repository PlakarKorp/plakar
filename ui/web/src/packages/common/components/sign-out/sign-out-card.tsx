import {
  Alert,
  Avatar,
  Button,
  Card,
  Typography,
} from "@plakar-ui/ui/components";
import { ReactNode } from "react";
import { LinkProps } from "react-aria-components";

type SignOutAccount = {
  name?: string;
  email?: string;
  avatarUrl?: string;
};

type SignOutCardProps = {
  productName: string;
  account?: SignOutAccount;
  stayHref: LinkProps["href"];
  isPending?: boolean;
  error?: ReactNode;
  onSignOut: () => void;
};

export function SignOutCard({
  productName,
  account,
  stayHref,
  isPending = false,
  error,
  onSignOut,
}: SignOutCardProps) {
  return (
    <Card className="mt-2 w-full max-w-md">
      <Card.HighlightedHeader
        illustration="waving"
        title="See you soon"
        description={`You are about to sign out of ${productName}.`}
      />

      <Card.Content padding="lg">
        <div className="flex flex-col gap-4">
          {account && (
            <div className="bg-secondary flex min-w-0 items-center gap-3 rounded-lg p-3">
              <Avatar
                text={account.name || account.email || ""}
                url={account.avatarUrl}
              />
              <div className="flex min-w-0 flex-col">
                <Typography size="sm" weight="medium" className="truncate">
                  {account.name || account.email}
                </Typography>
                {account.name && account.email && (
                  <Typography size="xs" className="text-secondary truncate">
                    {account.email}
                  </Typography>
                )}
              </div>
            </div>
          )}

          <Typography size="sm" className="text-secondary">
            Your data stays where it is and your backups keep running. Signing
            back in is all it takes to pick up where you left off.
          </Typography>

          {error && (
            <Alert variant="error" title="Unable to sign out" size="sm">
              {error}
            </Alert>
          )}

          <div className="flex flex-wrap justify-end gap-2">
            <Button variant="secondary" href={stayHref}>
              Stay signed in
            </Button>
            <Button
              variant="error"
              leftIcon="right-to-bracket"
              isLoading={isPending}
              onPress={onSignOut}
            >
              Sign out
            </Button>
          </div>
        </div>
      </Card.Content>
    </Card>
  );
}
