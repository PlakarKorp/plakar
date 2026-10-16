import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";

import { getGithubLoginUrl, logout, sendEmailLoginMagicLink } from "./api";

export function useSendEmailLoginMagicLink() {
  return useMutation({
    mutationFn: sendEmailLoginMagicLink,
  });
}

export function useGetGithubLoginUrl() {
  return useMutation({
    mutationFn: getGithubLoginUrl,
  });
}

export const useLogout = () => {
  const router = useRouter();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: logout,
    onSuccess: async () => {
      queryClient.removeQueries();
      await router.invalidate();
    },
  });
};
