import { CalendarDateTime, Time } from "@internationalized/date";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { useAppForm } from "../../hooks";
import { FormElement } from "../form";

function Template({ isReadOnly = false }: { isReadOnly?: boolean }) {
  const form = useAppForm({
    defaultValues: {
      name: "",
      description: "",
      role: "",
      comboRole: "",
      multiRole: [] as string[],
      tags: ["production"],
      categories: [] as string[],
      amount: 0,
      recurrence: "option1",
      notifications: false,
      date: null as CalendarDateTime | null,
      time: null as Time | null,
      excluded_from_coverage: false,
    },
    onSubmit: async ({ value }) => {
      await new Promise((r) => setTimeout(r, 1000));
      alert(`Submitted: ${JSON.stringify(value, null, 2)}`);
    },
  });

  const roleOptions = [
    { id: "owner", name: "Owner" },
    { id: "admin", name: "Admin" },
    { id: "member", name: "Member" },
  ];

  return (
    <FormElement
      handleSubmit={form.handleSubmit}
      isReadOnly={isReadOnly}
      className="flex max-w-sm flex-col gap-4"
    >
      <form.AppForm>
        <form.AppField
          name="name"
          validators={{
            onSubmit: ({ value }: { value: string }) =>
              value ? undefined : "Name is required",
          }}
        >
          {(field) => (
            <field.TextField
              label="Full Name"
              placeholder="e.g., Nestor Burma"
              supportText="Please enter your full name."
            />
          )}
        </form.AppField>

        <form.AppField
          name="description"
          validators={{
            onSubmit: ({ value }: { value: string }) =>
              value ? undefined : "Description is required",
          }}
        >
          {(field) => (
            <field.TextAreaField
              label="Description"
              placeholder="A few words about you…"
              supportText="Please enter a description."
            />
          )}
        </form.AppField>

        <form.AppField
          name="role"
          validators={{
            onSubmit: ({ value }: { value: string }) =>
              value ? undefined : "Role is required",
          }}
        >
          {(field) => (
            <field.SelectField
              label="Role"
              placeholder="Select a role"
              options={roleOptions}
              isClearable
              supportText="Type of account to configure"
            />
          )}
        </form.AppField>

        <form.AppField
          name="comboRole"
          validators={{
            onSubmit: ({ value }: { value: string }) =>
              value ? undefined : "Role is required",
          }}
        >
          {(field) => (
            <field.ComboBoxField
              label="Combo Role"
              placeholder="Select a group"
              options={roleOptions}
            />
          )}
        </form.AppField>

        <form.AppField
          name="multiRole"
          validators={{
            onSubmit: ({ value }: { value: string[] }) =>
              value.length ? undefined : "Role is required",
          }}
        >
          {(field) => (
            <field.MultiSelectField
              label="Multi Role"
              placeholder="Select roles"
              options={roleOptions}
            />
          )}
        </form.AppField>

        <form.AppField name="tags">
          {(field) => <field.TagsField label="Tags" tags={field.state.value} />}
        </form.AppField>

        <form.AppField
          name="categories"
          validators={{
            onSubmit: ({ value }: { value: string[] }) =>
              value.length ? undefined : "Categories are required",
          }}
        >
          {(field) => (
            <field.TagToggleGroupField
              label="Categories"
              selectionMode="multiple"
              supportText="Select all relevant categories."
              items={[
                { id: "compute", label: "Compute" },
                { id: "storage", label: "Block Storage" },
                { id: "database", label: "Database" },
                { id: "network", label: "Network" },
                { id: "security", label: "Security" },
              ]}
            />
          )}
        </form.AppField>

        <form.AppField
          name="amount"
          validators={{
            onSubmit: ({ value }: { value: number }) =>
              value > 0 ? undefined : "Amount must be greater than zero",
          }}
        >
          {(field) => (
            <field.NumberField
              label="Amount"
              minValue={0}
              maxValue={10}
              supportText="A number of items corresponding to the amount of stuff."
            />
          )}
        </form.AppField>

        <form.AppField
          name="date"
          validators={{
            onSubmit: ({ value }: { value: CalendarDateTime | null }) =>
              value ? undefined : "Date is required",
          }}
        >
          {(field) => <field.DateField label="Date" />}
        </form.AppField>

        <form.AppField
          name="time"
          validators={{
            onSubmit: ({ value }: { value: Time | null }) =>
              value ? undefined : "Time is required",
          }}
        >
          {(field) => <field.TimeField label="Time" inputWidth="w-28" />}
        </form.AppField>

        <form.AppField
          name="recurrence"
          validators={{
            onSubmit: ({ value }: { value: string }) =>
              value ? undefined : "recurrence is required",
          }}
        >
          {(field) => (
            <field.RadioGroupField
              label="Recurrence"
              items={[
                { value: "option1", label: "Twice daily" },
                { value: "option2", label: "Daily" },
                { value: "option3", label: "Weekly" },
                { value: "option4", label: "Every 30 days" },
              ]}
            />
          )}
        </form.AppField>

        <form.AppField name="notifications">
          {(field) => (
            <field.ToggleField
              title="Enable Notifications"
              description="Receive email notifications for important updates."
            />
          )}
        </form.AppField>

        <form.AppField name="excluded_from_coverage">
          {(field) => (
            <field.CheckboxField
              size="sm"
              label="Exclude from backup coverage"
              description="Excluded resources are omitted from protection status and coverage reporting."
            />
          )}
        </form.AppField>

        <div className="flex gap-2">
          <form.ResetButton className="flex-1">Reset</form.ResetButton>
          <form.SubmitButton className="flex-1">Submit</form.SubmitButton>
        </div>
      </form.AppForm>
    </FormElement>
  );
}

const meta = {
  title: "Components/Form",
  tags: ["autodocs"],
  component: Template,
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

/**
 * A read-only form reaches every field it contains. Inputs that have a
 * read-only rendering use it; the ones that don't -- checkbox, toggle, radio,
 * multi-select, tag toggles -- are disabled instead.
 */
export const ReadOnly: Story = {
  args: { isReadOnly: true },
};

const WithTextAreaFieldTemplate = () => {
  const form = useAppForm({
    defaultValues: {
      ssh_key: "",
    },
    onSubmit: async ({ value }) => {
      await new Promise((r) => setTimeout(r, 1000));
      alert(`Submitted: ${JSON.stringify(value, null, 2)}`);
    },
  });

  return (
    <FormElement
      handleSubmit={form.handleSubmit}
      className="flex max-w-lg flex-col gap-4"
    >
      <form.AppForm>
        <form.AppField
          name="ssh_key"
          validators={{
            onSubmit: ({ value }: { value: string }) =>
              value ? undefined : "SSH Key is required",
          }}
        >
          {(field) => (
            <field.TextAreaField
              label="SSH Key"
              placeholder="Begins with 'ssh-rsa', 'ecdsa-sha2-nistp256', 'ecdsa-sha2-nistp384', 'ecdsa-sha2-nistp521', 'ssh-ed25519', 'sk-ecdsa-sha2-nistp256@openssh.com', or 'sk-ssh-ed25519@openssh.com'"
              withFileImport
            />
          )}
        </form.AppField>

        <div className="flex gap-2">
          <form.ResetButton className="flex-1">Reset</form.ResetButton>
          <form.SubmitButton className="flex-1">Submit</form.SubmitButton>
        </div>
      </form.AppForm>
    </FormElement>
  );
};

export const TextAreaWithFileImport: Story = {
  render: () => <WithTextAreaFieldTemplate />,
};

const WithCodeEditorFieldTemplate = () => {
  const form = useAppForm({
    defaultValues: {
      json: "",
    },
    onSubmit: async ({ value }) => {
      await new Promise((r) => setTimeout(r, 1000));
      alert(`Submitted: ${JSON.stringify(value, null, 2)}`);
    },
  });

  return (
    <FormElement
      handleSubmit={form.handleSubmit}
      className="flex flex-col gap-4"
    >
      <form.AppForm>
        <form.AppField
          name="json"
          validators={{
            onSubmit: ({ value }: { value: string }) => {
              try {
                JSON.parse(value);
                return undefined;
              } catch {
                return "Invalid JSON format";
              }
            },
          }}
        >
          {(field) => (
            <field.CodeEditorField
              label="Google JSON Policy"
              placeholder="Enter your Google JSON policy"
              className="min-h-40"
              withFileImport
            />
          )}
        </form.AppField>

        <div className="flex gap-2">
          <form.ResetButton className="flex-1">Reset</form.ResetButton>
          <form.SubmitButton className="flex-1">Submit</form.SubmitButton>
        </div>
      </form.AppForm>
    </FormElement>
  );
};

export const CodeEditorWithFileImport: Story = {
  render: () => <WithCodeEditorFieldTemplate />,
};
