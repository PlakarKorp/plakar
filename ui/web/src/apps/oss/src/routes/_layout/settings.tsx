import { addToast, FormElement } from "@plakar-ui/common/components";
import { Main } from "@plakar-ui/common/layouts";
import { Tag, Toggle, Typography } from "@plakar-ui/ui/components";
import { useForm } from "@tanstack/react-form";
import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, useRouter } from "@tanstack/react-router";

import { useAccountServicesSetAlerting } from "~/api/account/mutations";
import { getAccountServicesAlertingQueryOptions } from "~/api/account/queries";
import type { AlertingSettings } from "~/api/account/schema";

export const Route = createFileRoute("/_layout/settings")({
  loader: async ({ context }) => {
    context.queryClient.ensureQueryData(
      getAccountServicesAlertingQueryOptions()
    );
  },
  component: RouteComponent,
});

function RouteComponent() {
  const alerting = useSuspenseQuery(getAccountServicesAlertingQueryOptions());
  const navigate = Route.useNavigate();
  const router = useRouter();

  if (alerting.data.__type === "unauthenticated") {
    navigate({
      to: "/login",
      search: { returnUrl: router.state.location.pathname },
    });
    return null;
  }
  return <SettingsForm alertingSettings={alerting.data.configuration} />;
}

type SettingsFormProps = {
  alertingSettings: AlertingSettings;
};

function SettingsForm({ alertingSettings }: SettingsFormProps) {
  const mutation = useAccountServicesSetAlerting();

  const form = useForm({
    defaultValues: {
      enabled: alertingSettings.enabled,
      email: alertingSettings.email_report,
    },
    onSubmit: (values) => {
      mutation.mutate(
        {
          enabled: values.value.enabled,
          email_report: values.value.email,
        },
        {
          onError: (error) => {
            form.reset();
            addToast({
              content: { children: error.toString(), variant: "error" },
            });
          },
        }
      );
    },
  });

  return (
    <Main title="Settings">
      <div className="flex flex-col gap-4">
        <Typography weight="medium" className="text-secondary">
          Alerting properties
        </Typography>

        <FormElement
          handleSubmit={form.handleSubmit}
          className="flex flex-col gap-4"
        >
          <form.Field
            name="enabled"
            children={(field) => (
              <Toggle
                aria-label={`${field.state.value ? "Disable" : "Enable"} alerting service at plakar.io`}
                title={
                  <>
                    Enable alerting service at{" "}
                    <strong className="text-accent">plakar.io</strong>
                  </>
                }
                description="Securely send concise task summaries to your account on our servers—your backup data stays on your side."
                isSelected={field.state.value}
                onChange={field.handleChange}
                onBlur={field.handleBlur}
              />
            )}
            listeners={{
              onChange: form.handleSubmit,
            }}
          />

          {alertingSettings.enabled && (
            <form.Field
              name="email"
              children={(field) => (
                <Toggle
                  aria-label={`${field.state.value ? "Disable" : "Receive"} alerts by email`}
                  title="Receive alerts by email"
                  description="Get issue notifications and weekly backup summaries—only metadata, never your actual data."
                  isSelected={field.state.value}
                  onChange={field.handleChange}
                  onBlur={field.handleBlur}
                />
              )}
              listeners={{
                onChange: form.handleSubmit,
              }}
            />
          )}

          <div className="flex items-start gap-2">
            <Toggle
              aria-label="Enable alerting to custom webhook"
              title="Enable alerting to custom webhook"
              description="Send alerts to your own webhook URL for custom processing."
              isDisabled
            />
            <Tag variant="brand" leftIcon="bell">
              Available soon
            </Tag>
          </div>
        </FormElement>
      </div>
    </Main>
  );
}
