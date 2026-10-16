import { Version } from "@plakar-ui/common/api/enrich";
import { z } from "zod";

export const ApiInfoSchema = z.object({
  authenticated: z.boolean(),
  version: z.string().transform((x) => Version(x)),
  browsable: z.boolean(),
  repository_id: z.string(),
  demo_mode: z.boolean().optional(),
});
