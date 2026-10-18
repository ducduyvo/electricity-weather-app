import Alert from "@mui/material/Alert";
import AppBar from "@mui/material/AppBar";
import Toolbar from "@mui/material/Toolbar";
import MuiButton from "@mui/material/Button";
import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import ToggleButton from "@mui/material/ToggleButton";
import ToggleButtonGroup from "@mui/material/ToggleButtonGroup";
import CircularProgress from "@mui/material/CircularProgress";
import Box from "@mui/material/Box";
import type { ButtonProps } from "@mui/material/Button";
import { useI18n } from "../../i18n";
import type { ReactNode } from "react";
import type { ForecastView } from "../../types/weather";

export function ApplicationShell({ children }: { children: ReactNode }) {
  const { t, locale, setLocale } = useI18n();
  return (
    <>
      <a className="skip-link" href="#forecast-workspace">
        {t("weather.skip_to_forecast")}
      </a>
      <AppBar position="static" elevation={0} className="app-bar">
        <Toolbar className="app-bar-content">
          <div className="app-brand">
            <svg viewBox="0 0 24 24" width="28" height="28" aria-hidden="true">
              <circle cx="12" cy="12" r="4" fill="currentColor" />
              <path
                d="M12 1v3m0 16v3M1 12h3m16 0h3M4 4l2 2m12 12 2 2M4 20l2-2M18 6l2-2"
                stroke="currentColor"
                strokeWidth="2"
              />
            </svg>
            <span>{t("weather.app_title")}</span>
          </div>
          <span className="app-bar-section">
            {t("weather.weather_outlook")}
          </span>
          <Select
            className="language-control"
            inputProps={{ "aria-label": t("weather.language") }}
            value={locale}
            onChange={(event) =>
              setLocale(event.target.value === "fi" ? "fi" : "en")
            }
          >
            <MenuItem value="en">English</MenuItem>
            <MenuItem value="fi">Suomi</MenuItem>
          </Select>
        </Toolbar>
      </AppBar>
      <div id="forecast-workspace">{children}</div>
    </>
  );
}

export function Button({ className = "", ...props }: ButtonProps) {
  return <MuiButton type="button" className={className} {...props} />;
}

export function ForecastSwitch({
  view,
  onChange,
}: {
  view: ForecastView;
  onChange: (view: ForecastView) => void;
}) {
  const { t } = useI18n();
  return (
    <ToggleButtonGroup
      className="forecast-switch"
      exclusive
      value={view}
      onChange={(_, next: ForecastView | null) => {
        if (next) onChange(next);
      }}
      aria-label={t("weather.forecast_resolution")}
      color="primary"
    >
      <ToggleButton value="hourly">{t("weather.hourly_view")}</ToggleButton>
      <ToggleButton value="daily">{t("weather.daily_view")}</ToggleButton>
    </ToggleButtonGroup>
  );
}

export function LoadingState() {
  const { t } = useI18n();
  return (
    <Box component="section" className="state" role="status" aria-live="polite">
      <CircularProgress size={20} aria-hidden="true" />
      {t("weather.loading_forecast")}
    </Box>
  );
}

export function EmptyForecastState() {
  const { t } = useI18n();
  return (
    <Alert severity="info" className="forecast-empty">
      {t("weather.no_daily_forecast_is_available_for_weather")}
    </Alert>
  );
}
