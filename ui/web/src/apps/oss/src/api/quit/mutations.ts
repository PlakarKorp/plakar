import { useMutation } from "@tanstack/react-query";

import { quit } from "./api";

export const useQuit = () =>
  useMutation({
    mutationFn: quit,
  });
