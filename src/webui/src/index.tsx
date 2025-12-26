/* @refresh reload */
import "./index.css";
import { render } from "solid-js/web";

import App from "./App";
import { themeService } from "./services/theme";
import { i18nService } from "./services/i18n";
import { defaultTheme } from "./themes/default";

const root = document.getElementById("root");

if (import.meta.env.DEV && !(root instanceof HTMLElement)) {
  throw new Error(
    "Root element not found. Did you forget to add it to your index.html? Or maybe the id attribute got misspelled?",
  );
}

// Initialize theme system
themeService.initialize();
themeService.loadTheme(defaultTheme);

// Initialize i18n system
i18nService.initialize();

render(() => <App />, root!);
