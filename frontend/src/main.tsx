import { ThemeProvider } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";
import { theme } from "./theme";
import { createRoot } from "react-dom/client";
import { I18nProvider } from "./i18n";
import App from "./App";
import "./style.css";

createRoot(document.getElementById("root")!).render(
  <ThemeProvider theme={theme}>
    <CssBaseline />
    <I18nProvider>
      <App />
    </I18nProvider>
  </ThemeProvider>,
);
