import type { StorybookConfig } from "@storybook/react-vite";

const config: StorybookConfig = {
  stories: ["../src/**/*.stories.@(ts|tsx)"],
  addons: [
    "@storybook/addon-essentials",
    "@storybook/addon-interactions",
    "@storybook/addon-themes",
  ],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
  viteFinal: (config) => {
    // Filter out the Sentry plugin — it warns about missing env vars
    // and isn't needed for Storybook.
    config.plugins = config.plugins?.filter((plugin) => {
      if (!plugin || Array.isArray(plugin)) return true;
      return (plugin as { name?: string }).name !== "sentry-vite-plugin";
    });
    return config;
  },
};
export default config;
