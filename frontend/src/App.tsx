import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import Alert from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import { useI18n, type TranslationKey } from "./i18n";
import { useEffect, useState } from "react";
import type { Forecast, ForecastView } from "./types/weather";
import { fetchForecast } from "./services/weather";
import { currentLocalHour, localDateLabel } from "./utils/forecastTime";
import {
  ApplicationShell,
  ForecastSwitch,
  LoadingState,
  Button,
} from "./components/common/Workspace";
import { DailyCards } from "./components/weather/DailyCards";
import { TradingInsights } from "./components/weather/TradingInsights";
import { ForecastViews } from "./components/weather/ForecastViews";

const cities = [
  ["Helsinki", 60.1699, 24.9384],
  ["Tampere", 61.4978, 23.761],
  ["Oulu", 65.0121, 25.4651],
  ["Stockholm", 59.3293, 18.0686],
] as const;
export default function App() {
  const { t, locale } = useI18n();
  const [city, setCity] = useState(0);
  const [data, setData] = useState<Forecast | null>(null);
  const [error, setError] = useState<TranslationKey | "">("");
  const [loading, setLoading] = useState(true);
  const [retry, setRetry] = useState(0);
  const [view, setView] = useState<ForecastView>("hourly");
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError("");
    setData(null);
    const [, lat, lon] = cities[city];
    fetchForecast(lat, lon, controller.signal)
      .then((forecast) => {
        if (!controller.signal.aborted) setData(forecast);
      })
      .catch(() => {
        if (!controller.signal.aborted) {
          setError("weather.weather_unavailable");
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [city, retry]);
  const cutoff = data ? currentLocalHour(data.timezone || "UTC") : "";
  const chart = data?.hourly
    .filter((point) => point.time >= cutoff)
    .slice(0, 72)
    .map((point) => ({ ...point, label: localDateLabel(point.time, true, locale) }));
  const dailyChart = data?.daily.slice(0, 14).map((day) => ({
    ...day,
    label: localDateLabel(day.date, false, locale),
    // Use the backend completeness flag; never plot missing/partial energy as zero.
    solarEnergy:
      day.completeDay && day.solarSampleCount === day.hourCount
        ? day.solarEnergyKWhM2
        : null,
  }));
  const activeSignals =
    view === "hourly" ? data?.signals : data?.outlookSignals;
  return (
    <ApplicationShell>
      <main>
        <header className="page-header">
          <div>
            <span className="eyebrow">{t("weather.page_eyebrow")}</span>
            <Typography component="h1" variant="h1">{t("weather.app_title")}</Typography>
            <p>{t("weather.page_description")}</p>
          </div>
          <label className="city-control">
            <span>{t("weather.forecast_location")}</span>
            <Select
              inputProps={{ "aria-label": t("weather.forecast_location") }}
              value={city}
              onChange={(e) => setCity(+e.target.value)}
            >
              {cities.map((c, i) => (
                <MenuItem value={i} key={c[0]}>
                  {c[0] === "Stockholm" ? t("weather.stockholm") : c[0]}
                </MenuItem>
              ))}
            </Select>
          </label>
        </header>
        {loading && <LoadingState />}
        {error && (
          <Alert severity="error" className="forecast-error">
            {t(error)}{" "}
            <Button onClick={() => setRetry((value) => value + 1)}>{t("weather.retry")}</Button>
          </Alert>
        )}
        {data && chart && (
          <>
            <ForecastSwitch view={view} onChange={setView} />
            <p className="forecast-caption">
              {view === "hourly"
                ? t("weather.hourly_detail_for_short_term_operational_analysis")
                : t("weather.daily_aggregates_for_broader_weather_trends")}{" "}
              {t("weather.times_and_dates")} {data.timezone}.
            </p>
            <DailyCards data={data} />
            <TradingInsights view={view} itemsForView={activeSignals} />
            <ForecastViews
              view={view}
              data={data}
              chart={chart}
              dailyChart={dailyChart}
            />
            <footer>
              <p>{t("weather.electricity_retail_weather_indicators")}</p>
              {t("weather.timezone")} {data.timezone} · {t("weather.forecast_source")} Open-Meteo · {t("weather.updated")}{" "}
              {new Date(data.fetchedAt).toLocaleTimeString(locale)}
            </footer>
          </>
        )}
      </main>
    </ApplicationShell>
  );
}
