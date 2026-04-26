import type { StorybookConfig } from "@storybook/react-vite";

const config: StorybookConfig = {
  stories: ["../src/**/*.stories.@(ts|tsx)"],
  addons: ["@storybook/addon-docs", "@storybook/addon-themes"],
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
      allowedHosts: true,
      // Fix HMR WebSocket when accessed via ngrok/tunnel:
      // Override the WS URL the browser uses so it connects back via wss on the public host.
      // clientPort tells Vite what port to advertise to the browser (443 = ngrok HTTPS).
      hmr:
        config.server?.hmr !== false
          ? {
              ...(typeof config.server?.hmr === "object"
                ? config.server.hmr
                : {}),
              ...(process.env.STORYBOOK_HMR_HOST
                ? {
                    protocol: "wss",
                    clientPort: 443,
                    path: "/@vite/client",
                  }
                : {}),
            }
          : false,
    };
    return config;
  },
};
export default config;
