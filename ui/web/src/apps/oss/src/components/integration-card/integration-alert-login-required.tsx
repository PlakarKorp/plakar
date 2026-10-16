import { Alert, Button, Terminal } from "@plakar-ui/ui/components";

type IntegrationAlertLoginRequiredProps = {
  name?: string;
  version?: string;
  os?: string;
  arch?: string;
};

export function IntegrationAlertLoginRequired({
  name = "<name>",
  version = "<version>",
  os = "<os>",
  arch = "<arch>",
}: IntegrationAlertLoginRequiredProps) {
  return (
    <Alert
      title="Login is required to install pre-built integrations"
      variant="warning"
      actions={
        <Button
          variant="secondary"
          href={{
            to: "/login",
            search: {
              returnUrl: "/integrations",
            },
          }}
        >
          Login
        </Button>
      }
    >
      <p>
        To make your life easier, we host pre-built integrations for various
        systems that can be installed in one click directly from the UI (login
        required).
      </p>
      <p className="mt-2">
        Alternatively, you can build and install integrations manually with the
        CLI.
      </p>

      <div className="mt-2">
        <Terminal title="Build & Install integration with the CLI">
          <Terminal.Command prompt="$ " command={`plakar pkg build ${name}`} />
          <Terminal.Command
            prompt="$ "
            command={`plakar pkg add ./${name}_${version}_${os}_${arch}.ptar`}
          />
        </Terminal>
      </div>
    </Alert>
  );
}
