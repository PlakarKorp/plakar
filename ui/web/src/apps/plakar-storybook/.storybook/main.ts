import type { StorybookConfig } from "@storybook/react-vite";

const config: StorybookConfig = {
  stories: [
    "../../../packages/common/components/**/*.stories.@(js|jsx|mjs|ts|tsx)",
    "../../../packages/ui/components/**/*.stories.@(js|jsx|mjs|ts|tsx)",
    "../../../apps/oss/src/components/**/*.stories.@(js|jsx|mjs|ts|tsx)",
  ],
  addons: ["@storybook/addon-docs"],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
};
export default config;
