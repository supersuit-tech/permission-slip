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
  core: {
    // Allow external hosts (e.g. ngrok tunnels for remote preview)
    allowedHosts: true as unknown as string[],
  },
  viteFinal: (config) => {
    // Filter out the Sentry plugin — it warns about missing env vars
    // and isn't needed for Storybook.
    config.plugins = config.plugins?.filter((plugin) => {
      if (!plugin || Array.isArray(plugin)) return true;
      return (plugin as { name?: string }).name !== "sentry-vite-plugin";
    });
    // Allow ngrok and other external hosts for the preview iframe
    config.server = {
      ...config.server,
      allowedHosts: "all",
    };
    return config;
  },
};
export default config;
