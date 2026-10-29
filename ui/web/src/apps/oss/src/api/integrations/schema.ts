import { Date } from "@plakar-ui/common/api/enrich";
import { PaginationParams } from "@plakar-ui/common/api/pagination";
import { z } from "zod";

export const IntegrationTypeSchema = z.enum([
  "storage",
  "source",
  "destination",
]);

export type IntegrationType = z.infer<typeof IntegrationTypeSchema>;

const IntegrationInstallationStatusSchema = z.enum([
  "installed",
  "not-installed",
  "built-in",
]);

export const IntegrationInstallationStatusFilterSchema = z.enum([
  ...IntegrationInstallationStatusSchema.options,
  "upgradable",
]);

export type IntegrationInstallationStatusFilter = z.infer<
  typeof IntegrationInstallationStatusFilterSchema
>;

export const IntegrationSchema = z.object({
  id: z.string(),
  name: z.string(),
  display_name: z.string(),
  description: z.string(),
  documentation: z.string(),
  icon: z.union([z.string().url(), z.literal("")]),
  featured: z.union([z.string().url(), z.literal("")]),
  tags: z.array(z.string()),
  latest_version: z.string(),
  stage: z.string(),
  types: z.object({
    storage: z.boolean(),
    source: z.boolean(),
    destination: z.boolean(),
  }),
  installation: z.object({
    status: IntegrationInstallationStatusSchema,
    // XXX: absent if not-installed, fix the type
    version: z.string().optional(),
    available: z.boolean(),
  }),
});

export type Integration = z.infer<typeof IntegrationSchema>;

export const IntegrationsListSchema = z.object({
  total: z.number(),
  items: z.array(IntegrationSchema),
});

export type ListIntegrationsParams = PaginationParams & {
  search?: string;
  types?: IntegrationType[];
  installationStatus?: IntegrationInstallationStatusFilter;
};

const IntegrationMessageSchema = z.object({
  date: z.string().transform(Date),
  message: z.string(),
});

export const IntegrationOperationResultSchema = z.object({
  status: z.enum(["ok", "failed"]),
  messages: z.array(IntegrationMessageSchema),
});
