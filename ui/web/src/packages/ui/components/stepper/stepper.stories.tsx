import { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Button, Checkbox, Select, TextInput, Toggle, Typography } from "..";
import { Stepper } from "./stepper";

const meta: Meta<typeof Stepper> = {
  title: "design-system/Stepper",
  component: Stepper,
  tags: ["autodocs"],
  argTypes: {},
};
export default meta;

type Story = StoryObj<typeof Stepper>;

export const Default: Story = {
  render: () => (
    <div className="m-auto w-[48rem]">
      <Stepper
        steps={
          [
            "Authentication",
            "Configuration",
            "Settings",
            "Review & Create",
          ] as const
        }
      >
        {({ currentStep, goToPreviousStep, goToNextStep }) => (
          <div className="grid w-full grid-cols-2 justify-center gap-4">
            <Button
              size="md"
              variant="secondary"
              leftIcon="arrow-left"
              onPress={goToPreviousStep}
              className={"w-full justify-self-end"}
              isDisabled={currentStep === "Authentication"}
            >
              Back
            </Button>
            <Button
              size="md"
              variant="primary"
              rightIcon="arrow-right"
              onPress={goToNextStep}
              className="w-full"
              isDisabled={currentStep === "Review & Create"}
            >
              Next
            </Button>
          </div>
        )}
      </Stepper>
    </div>
  ),
};

export const InteractiveWizard: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: () => <WizardTemplate />,
};

function WizardTemplate() {
  const [auth, setAuth] = useState({ company: "", email: "", phone: "" });
  const getAuthProgress = ({
    company,
    email,
    phone,
  }: {
    company: string;
    email: string;
    phone: string;
  }) =>
    (company === "" ? 0 : 33) +
    (email === "" ? 0 : 33) +
    (phone === "" ? 0 : 34);

  const [config, setConfig] = useState({ requiredChoice: "" }); // selection id
  const [settings, setSettings] = useState({
    flagA: false,
    flagB: false,
    toggleNote: false,
  });

  return (
    <div className="m-auto w-[52rem]">
      <Stepper
        steps={
          [
            "Authentication",
            "Configuration",
            "Settings",
            "Review & Create",
          ] as const
        }
      >
        {({ currentStep, setProgress, goToPreviousStep, goToNextStep }) =>
          ({
            Authentication: (
              <div className="flex flex-col gap-8">
                <div className="m-auto flex max-w-xl flex-col gap-6">
                  <TextInput
                    label="Company Name"
                    value={auth.company}
                    onChange={(v) => {
                      setAuth((s) => ({ ...s, company: v }));
                      setProgress(
                        getAuthProgress({
                          company: v,
                          email: auth.email,
                          phone: auth.phone,
                        })
                      );
                    }}
                  />
                  <div className="flex w-full gap-4">
                    <TextInput
                      label="Email Address"
                      className="w-full"
                      value={auth.email}
                      onChange={(v) => {
                        setAuth((s) => ({ ...s, email: v }));
                        setProgress(
                          getAuthProgress({
                            company: auth.company,
                            email: v,
                            phone: auth.phone,
                          })
                        );
                      }}
                    />
                    <TextInput
                      label="Phone number"
                      className="w-full"
                      placeholder="+1 555 555 5555"
                      value={auth.phone}
                      onChange={(v) => {
                        setAuth((s) => ({ ...s, phone: v }));
                        setProgress(
                          getAuthProgress({
                            company: auth.company,
                            email: auth.email,
                            phone: v,
                          })
                        );
                      }}
                    />
                  </div>
                </div>
                <div className="grid w-full grid-cols-2 justify-center gap-4">
                  <Button
                    size="md"
                    variant="primary"
                    rightIcon="arrow-right"
                    onPress={goToNextStep}
                    className="col-start-2 w-full"
                    isDisabled={
                      auth.company === "" ||
                      auth.email === "" ||
                      auth.phone === ""
                    }
                  >
                    Next
                  </Button>
                </div>
              </div>
            ),
            Configuration: (
              <div className="flex flex-col gap-8">
                <div className="m-auto flex max-w-xl flex-col gap-6">
                  <Select
                    label="Some mandatory setting"
                    placeholder="Select an option"
                    value={config.requiredChoice || null}
                    onChange={(k) => setConfig({ requiredChoice: String(k) })}
                  >
                    <Select.ListBox>
                      <Select.Item id="option-1" name="Option A" />
                      <Select.Item id="option-2" name="Option B" />
                      <Select.Item id="option-3" name="Option C" />
                    </Select.ListBox>
                  </Select>
                </div>
                <div className="grid w-full grid-cols-2 justify-center gap-4">
                  <Button
                    size="md"
                    variant="secondary"
                    leftIcon="arrow-left"
                    onPress={goToPreviousStep}
                    className={"w-full justify-self-end"}
                  >
                    Back
                  </Button>
                  <Button
                    size="md"
                    variant="primary"
                    rightIcon="arrow-right"
                    onPress={goToNextStep}
                    isDisabled={config.requiredChoice === ""}
                    className="w-full"
                  >
                    Next
                  </Button>
                </div>
              </div>
            ),
            Settings: (
              <div className="flex flex-col gap-8">
                <div className="m-auto flex max-w-xl flex-col gap-6">
                  <div className="flex flex-col gap-3">
                    <Checkbox
                      isSelected={settings.flagA}
                      onChange={(v) => setSettings((s) => ({ ...s, flagA: v }))}
                      label="Enable feature A"
                    />
                    <Checkbox
                      isSelected={settings.flagB}
                      onChange={(v) => setSettings((s) => ({ ...s, flagB: v }))}
                      label="Enable feature B"
                    />
                  </div>
                  <Toggle
                    title="Plain text"
                    description="Enable additional logging"
                    isSelected={settings.toggleNote}
                    onChange={(v) =>
                      setSettings((s) => ({ ...s, toggleNote: v }))
                    }
                  />
                </div>
                <div className="grid w-full grid-cols-2 justify-center gap-4">
                  <Button
                    size="md"
                    variant="secondary"
                    leftIcon="arrow-left"
                    onPress={goToPreviousStep}
                    className={"w-full justify-self-end"}
                  >
                    Back
                  </Button>
                  <Button
                    size="md"
                    variant="primary"
                    rightIcon="arrow-right"
                    onPress={goToNextStep}
                    className="w-full"
                  >
                    Next
                  </Button>
                </div>
              </div>
            ),
            "Review & Create": (
              <ReviewSummary auth={auth} config={config} settings={settings} />
            ),
          })[currentStep]
        }
      </Stepper>
    </div>
  );
}

function ReviewSummary({
  auth,
  config,
  settings,
}: {
  auth: { company: string; email: string; phone: string };
  config: { requiredChoice: string };
  settings: { flagA: boolean; flagB: boolean; toggleNote: boolean };
}) {
  return (
    <div className="m-auto flex max-w-xl flex-col gap-6">
      <Typography weight="medium" className="text-primary">
        Summary
      </Typography>
      <ul className="text-secondary ml-5 list-disc">
        <li>Company: {auth.company || "—"}</li>
        <li>Email: {auth.email || "—"}</li>
        <li>Phone: {auth.phone || "—"}</li>
        <li>Mandatory setting: {config.requiredChoice || "—"}</li>
        <li>Feature A: {settings.flagA ? "Enabled" : "Disabled"}</li>
        <li>Feature B: {settings.flagB ? "Enabled" : "Disabled"}</li>
        <li>
          Plain text logging: {settings.toggleNote ? "Enabled" : "Disabled"}
        </li>
      </ul>
    </div>
  );
}

export const Segmented: Story = {
  render: () => (
    <div className="m-auto w-[48rem]">
      <Stepper steps={["Rules", "Scope"] as const} variant="segmented">
        {({ currentStep, setProgress, goToPreviousStep, goToNextStep }) => (
          <div className="flex flex-col gap-4">
            <div className="flex gap-2">
              {[0, 50, 100].map((value) => (
                <Button
                  key={value}
                  size="md"
                  variant="secondary"
                  onPress={() => setProgress(value)}
                >
                  {`${value}%`}
                </Button>
              ))}
            </div>
            <div className="grid w-full grid-cols-2 gap-4">
              <Button
                size="md"
                variant="secondary"
                leftIcon="arrow-left"
                onPress={goToPreviousStep}
                isDisabled={currentStep === "Rules"}
              >
                Back
              </Button>
              <Button
                size="md"
                variant="primary"
                rightIcon="arrow-right"
                onPress={goToNextStep}
                isDisabled={currentStep === "Scope"}
              >
                Next
              </Button>
            </div>
          </div>
        )}
      </Stepper>
    </div>
  ),
};
