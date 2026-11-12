import { z } from "zod";

export const RepositoryFacetsSchema = z.object({
  tags: z.array(z.string()),
  paths: z.array(z.string()),
});
