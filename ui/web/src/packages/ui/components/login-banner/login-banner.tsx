import { Link, LinkProps } from "react-aria-components";

import IllustrationHello from "@plakar-ui/ui/assets/plakar-illustration-hello.svg";

import { Button, Typography } from "..";

type LoginBannerProps = Pick<LinkProps, "href"> & {
  onClick?: () => void;
};

export function LoginBanner({ href, onClick }: LoginBannerProps) {
  const withLoginButton = !href;

  return (
    <div className="bg-accent grid grid-cols-[1fr_auto] items-center rounded-sm">
      <div className="flex items-center justify-center gap-1">
        <div className="flex flex-col justify-center gap-0.5 py-2 pl-3">
          <Typography className="text-on-brand" weight="medium" size="sm">
            {withLoginButton ? (
              "Sign in"
            ) : (
              <Link
                href={href}
                onPress={onClick}
                className="hover:text-on-brand/80 underline"
              >
                Sign in
              </Link>
            )}{" "}
            to get access to all features
          </Typography>
          <Typography className="text-on-brand" size="xs">
            Magic link or GitHub
          </Typography>
        </div>

        <div className="self-end">
          <img
            alt="plakar.io"
            src={IllustrationHello}
            className="h-[70px] w-[90px]"
          />
        </div>
      </div>
      {withLoginButton && (
        <Button
          leftIcon="right-to-bracket"
          variant="accent-tertiary"
          className="mr-2 sm:mr-6"
          onClick={onClick}
          slot="close"
        >
          Sign in
        </Button>
      )}
    </div>
  );
}
