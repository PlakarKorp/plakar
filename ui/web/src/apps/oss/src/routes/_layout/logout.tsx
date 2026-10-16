import { APIResponseBadRequest } from "@plakar-ui/common/api/errors";
import { SignOutCard } from "@plakar-ui/common/components";
import { Main } from "@plakar-ui/common/layouts";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useRouter } from "@tanstack/react-router";

import { getAccountMeQueryOptions } from "~/api/account/queries";
import { useLogout } from "~/api/auth/mutations";

export const Route = createFileRoute("/_layout/logout")({
  component: Component,
});

function Component() {
  const router = useRouter();
  const mutation = useLogout();
  const account = useQuery(getAccountMeQueryOptions());

  const doLogout = async () => {
    await mutation.mutateAsync();
    router.history.push("/");
  };

  return (
    <Main>
      <div className="flex h-full items-center justify-center py-4">
        <SignOutCard
          productName="Plakar"
          account={
            account.data
              ? {
                  name: account.data.name,
                  avatarUrl: account.data.avatar_url,
                }
              : undefined
          }
          stayHref={{ to: "/" }}
          isPending={mutation.isPending}
          error={
            mutation.error instanceof APIResponseBadRequest
              ? mutation.error.responsePayload
              : mutation.error?.message
          }
          onSignOut={doLogout}
        />
      </div>
    </Main>
  );
}
