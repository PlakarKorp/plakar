import { getLocalStorage, setLocalStorage } from "@plakar-ui/common/utils";

export const PlakarAPITokenLocalStorage = {
  store: (token: string) => {
    setLocalStorage("plakar_token", token);
  },

  get: () => {
    return getLocalStorage<string>("plakar_token");
  },
};
