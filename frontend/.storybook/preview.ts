import type { Preview } from "@storybook/react";
import { withThemeByClassName } from "@storybook/addon-themes";
import "../src/index.css";

const preview: Preview = {
  decorators: [
    withThemeByClassName({
      themes: {
        light: "",
        dark: "dark",
      },
      defaultTheme: "light",
      parentSelector: "html",
    }),
  ],
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    layout: "centered",
  },
};
export default preview;
