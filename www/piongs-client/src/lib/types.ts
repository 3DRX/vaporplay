import { z } from "zod";

export const formSchema = z.object({
  server: z.string().nonempty(),
});

export type FormType = z.infer<typeof formSchema>;

