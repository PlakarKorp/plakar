import { z } from "zod";

export const RepositoryFacetsSchema = z.object({
  tags: z.array(z.string()),
  paths: z.array(z.string()),
});

export const IntegrityFailureSchema = z.object({
  path: z.string(),
  error: z.string(),
});

export type IntegrityFailure = z.infer<typeof IntegrityFailureSchema>;

/** The verification of a snapshot: running until every file was read back. */
export const SnapshotIntegritySchema = z.object({
  item: z.object({
    status: z.enum(["running", "done", "failed"]),
    checked: z.number(),
    corrupted: z.array(IntegrityFailureSchema),
    message: z.string().optional(),
  }),
});

export type SnapshotIntegrity = z.infer<typeof SnapshotIntegritySchema>["item"];
