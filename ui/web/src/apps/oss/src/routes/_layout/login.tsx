import {
  addToast,
  FormElement,
  requiredEmailValidators,
  requiredValidators,
} from "@plakar-ui/common/components";
import { useAppForm } from "@plakar-ui/common/hooks";
import { Main } from "@plakar-ui/common/layouts";
import {
  Alert,
  Button,
  Card,
  Link,
  TextInput,
  Typography,
} from "@plakar-ui/ui/components";
import { useForm } from "@tanstack/react-form";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { z } from "zod";

import {
  useGetGithubLoginUrl,
  useSendEmailLoginMagicLink,
} from "~/api/auth/mutations";
import { pollLoginQueryOptions } from "~/api/auth/queries";
import { LoginStartResponse } from "~/api/auth/schema";
import { useWaitLoginStatusThenRedirect } from "~/hooks/use-api";

export const Route = createFileRoute("/_layout/login")({
  validateSearch: z.object({
    returnUrl: z.string().default("").catch(""),
  }),
  component: RouteComponent,
});

type PendingLogin = LoginStartResponse & {
  provider: "github" | "email";
  email?: string;
};

const toastError = (error: unknown) =>
  addToast({
    content: {
      children: error instanceof Error ? error.message : "An error occurred",
      variant: "error",
    },
  });

function RouteComponent() {
  const { returnUrl } = Route.useSearch();
  const getGithubLoginUrlMutation = useGetGithubLoginUrl();
  const sendEmailLoginMagicLinkMutation = useSendEmailLoginMagicLink();
  const [pendingLogin, setPendingLogin] = useState<PendingLogin | null>(null);

  const form = useForm({
    defaultValues: {
      email: "",
    },
    onSubmit: async ({ value }) => {
      sendEmailLoginMagicLinkMutation.mutate(
        { email: value.email },
        {
          onSuccess: (res) =>
            setPendingLogin({ ...res, provider: "email", email: value.email }),
          onError: toastError,
        }
      );
    },
  });

  const startGithubLogin = () => {
    // Open the tab while still handling the click, or the browser blocks it
    // as a popup; it is pointed at the sign-in page once the backend answers.
    const signInTab = window.open("", "_blank");
    getGithubLoginUrlMutation.mutate(undefined, {
      onSuccess: (res) => {
        if (signInTab) {
          signInTab.opener = null;
          signInTab.location.href = res.URL;
        }
        setPendingLogin({ ...res, provider: "github" });
      },
      onError: (error) => {
        signInTab?.close();
        toastError(error);
      },
    });
  };

  const startOver = () => {
    setPendingLogin(null);
    sendEmailLoginMagicLinkMutation.reset();
    getGithubLoginUrlMutation.reset();
    form.reset();
  };

  useWaitLoginStatusThenRedirect({ status: "logged_in", returnUrl });

  return (
    <Main
      title="Sign in to protect your backups"
      description="Set up instant alerts to catch backup issues early — and get access to new features as we roll them out."
    >
      <Card className="mt-10 sm:mx-auto sm:w-full sm:max-w-[480px]">
        <Card.Content padding="xl">
          {pendingLogin ? (
            <CompleteLogin login={pendingLogin} onStartOver={startOver} />
          ) : (
            <>
              <FormElement
                handleSubmit={form.handleSubmit}
                className="space-y-6"
              >
                <form.Field
                  name="email"
                  validators={requiredEmailValidators}
                  children={(field) => (
                    <TextInput
                      label="Email address"
                      placeholder="nestor@plakar.io"
                      isRequired
                      isInvalid={field.state.meta.errors.length > 0}
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={field.handleChange}
                    />
                  )}
                />

                <form.Subscribe
                  selector={(state) => [state.canSubmit, state.isSubmitting]}
                  children={([canSubmit, isSubmitting]) => (
                    <Button
                      type="submit"
                      variant="primary"
                      isDisabled={!canSubmit}
                      className="w-full"
                      isPending={
                        isSubmitting ||
                        sendEmailLoginMagicLinkMutation.isPending
                      }
                    >
                      Send magic link
                    </Button>
                  )}
                />
              </FormElement>

              <div>
                <div className="relative my-8">
                  <div
                    aria-hidden="true"
                    className="absolute inset-0 flex items-center"
                  >
                    <div className="w-full border-t border-gray-200" />
                  </div>
                  <div className="relative flex justify-center text-sm/6 font-medium">
                    <span className="text-primary bg-primary px-6">
                      Or continue with
                    </span>
                  </div>
                </div>

                <Button
                  className="w-full"
                  leftIcon="github"
                  variant="secondary"
                  onPress={startGithubLogin}
                  isPending={getGithubLoginUrlMutation.isPending}
                >
                  GitHub
                </Button>
              </div>
            </>
          )}
        </Card.Content>
      </Card>
    </Main>
  );
}

type CompleteLoginProps = {
  login: PendingLogin;
  onStartOver: () => void;
};

// The sign-in page shows a code only to whoever completed the sign-in; typing
// it here proves this plakar is theirs, not someone who lured them into a link.
function CompleteLogin({ login, onStartOver }: CompleteLoginProps) {
  const [completionCode, setCompletionCode] = useState("");

  const pollQuery = useQuery({
    ...pollLoginQueryOptions({
      pollId: login.poll_id,
      completionCode: completionCode || undefined,
    }),
    // A new code is a new query key; keep the prompt on screen meanwhile.
    placeholderData: (previous) => previous,
    // Once the sign-in is complete, polling again without a new code cannot
    // change the answer.
    refetchInterval: (query) =>
      query.state.data?.status === "pending" ? 2000 : false,
  });

  // An expired or unknown sign-in cannot be completed: go back to the start.
  const { isError, error } = pollQuery;
  useEffect(() => {
    if (isError) {
      toastError(error);
      onStartOver();
    }
  }, [isError, error, onStartOver]);

  const form = useAppForm({
    defaultValues: {
      code: "",
    },
    onSubmit: ({ value }) => {
      const code = value.code.trim();
      if (code === completionCode) {
        void pollQuery.refetch();
        return;
      }
      setCompletionCode(code);
    },
  });

  const status = pollQuery.data?.status;

  return (
    <form.AppForm>
      <FormElement handleSubmit={form.handleSubmit} className="space-y-6">
        {status === "invalid-code" && (
          <Alert variant="error" title="Wrong confirmation code">
            This code does not match. Check the code on the sign-in page and try
            again.
          </Alert>
        )}

        <Typography weight="semibold" className="text-primary text-center">
          {login.provider === "email" ? (
            <>Check {login.email} for a magic link to sign in.</>
          ) : (
            <>
              Sign in with GitHub in the{" "}
              <Link href={login.URL} target="_blank" rel="noopener noreferrer">
                sign-in page
              </Link>
              .
            </>
          )}
          <br />
          Then enter the confirmation code it shows here.
        </Typography>

        <form.AppField name="code" validators={requiredValidators}>
          {(field) => (
            <field.TextField
              label="Confirmation code"
              placeholder="M4TP-K7WQ"
              autoComplete="one-time-code"
              isRequired
            />
          )}
        </form.AppField>

        <form.SubmitButton
          className="w-full"
          isPending={
            (pollQuery.isFetching && !!completionCode) || status === "ok"
          }
        >
          Confirm
        </form.SubmitButton>

        <Typography size="sm" className="text-center">
          <Link onPress={onStartOver}>Start over</Link>
        </Typography>
      </FormElement>
    </form.AppForm>
  );
}
