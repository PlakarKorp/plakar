import { ThemeContext } from "@plakar-ui/common/theme";
import { Preview } from "@storybook/react-vite";
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from "@tanstack/react-router";

import "./index.css";

const preview: Preview = {
  // Disable the default button to select the preview background
  parameters: {
    backgrounds: { disable: true },
  },

  globalTypes: {
    theme: {
      description: "Global theme for components",
      toolbar: {
        title: "Theme",
        icon: "circlehollow",
        items: [
          { value: "light", icon: "circlehollow", title: "light" },
          { value: "dark", icon: "circle", title: "dark" },
          { value: "side-by-side", icon: "sidebar", title: "side by side" },
        ],
        // Change title based on selected value
        dynamicTitle: true,
      },
    },
  },
  initialGlobals: {
    theme: "light",
  },
  decorators: [
    (Story, context) => {
      const theme = context.globals.theme;

      return (
        <RouterProvider
          router={createRouter({
            history: createMemoryHistory(),
            routeTree: createRootRoute({
              component: () => (
                <div className="flex flex-col gap-4">
                  {["light", "side-by-side"].includes(theme) && (
                    <ThemeContext.Provider
                      value={{
                        value: "light",
                        mode: "light",
                        setMode: () => {},
                      }}
                    >
                      <div className="rounded-sm bg-white p-4">
                        <Story />
                      </div>
                    </ThemeContext.Provider>
                  )}

                  {["dark", "side-by-side"].includes(theme) && (
                    <div className="text-primary dark rounded-sm bg-slate-950 p-4">
                      <ThemeContext.Provider
                        value={{
                          value: "dark",
                          mode: "dark",
                          setMode: () => {},
                        }}
                      >
                        <Story />
                      </ThemeContext.Provider>
                    </div>
                  )}
                </div>
              ),
            }),
          })}
        />
      );
    },
  ],
};

export default preview;
