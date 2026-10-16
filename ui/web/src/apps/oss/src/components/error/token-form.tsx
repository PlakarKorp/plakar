import { FormElement, requiredValidators } from "@plakar-ui/common/components";
import { useAppForm } from "@plakar-ui/common/hooks";
import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import z from "zod";

import { PlakarAPITokenLocalStorage } from "~/api/plakar/localstorage";

export function TokenForm() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const form = useAppForm({
    defaultValues: {
      token: "",
    },
    onSubmit: ({ value }) => {
      PlakarAPITokenLocalStorage.store(value.token);
      // Invalidate react-query cache
      queryClient.removeQueries();
      // Invalidate the router
      router.invalidate();
    },
  });

  return (
    <form.AppForm>
      <FormElement
        handleSubmit={form.handleSubmit}
        className="flex flex-col items-start gap-2 md:flex-row"
      >
        <form.AppField
          name="token"
          validators={{
            ...requiredValidators,
            onSubmit: ({ value }) => {
              const err = requiredValidators.onSubmit({ value });
              if (err) {
                return err;
              }

              const parsed = z.uuid().safeParse(value);
              if (!parsed.success) {
                return "Invalid token format.";
              }
            },
          }}
        >
          {(field) => (
            <field.TextField
              label="API Token"
              icon="key"
              type="password"
              autoComplete="off"
              autoCorrect="off"
              placeholder="UUID"
              className="md:w-sm"
              isRequired
            />
          )}
        </form.AppField>

        <form.SubmitButton className="mt-7">Sign in</form.SubmitButton>
      </FormElement>
    </form.AppForm>
  );
}
