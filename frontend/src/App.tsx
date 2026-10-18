import Select from "@mui/material/Select";
import FormControl from "@mui/material/FormControl";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Alert from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import { useI18n, type TranslationKey } from "./i18n";
import { useEffect, useState } from "react";
import type { Forecast, ForecastView } from "./types/weather";
import { fetchForecast } from "./services/weather";
import { dailyChartValues } from "./utils/dailyForecast";
import {
  currentLocalHour,
  localDateLabel,
  currentForecastHour,
  hourlyPointLabel,
} from "./utils/forecastTime";
import {
  ApplicationShell,
  ForecastSwitch,
  LoadingState,
  EmptyForecastState,
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
  const instantCutoff = data ? currentForecastHour(data.timezone || "UTC") : 0;
  const timeOccurrences = new Map<string, number>();
  for (const point of data?.hourly ?? [])
    timeOccurrences.set(point.time, (timeOccurrences.get(point.time) ?? 0) + 1);
  const chart = data?.hourly
    .filter((point) =>
      point.unixTime != null
        ? point.unixTime >= instantCutoff
        : point.time >= cutoff,
    )
    .slice(0, 72)
    .map((point) => ({
      ...point,
      label:
        point.unixTime != null
          ? hourlyPointLabel(
              point.unixTime,
              data!.timezone || "UTC",
              locale,
              (timeOccurrences.get(point.time) ?? 0) > 1,
            )
          : localDateLabel(point.time, true, locale),
    }));
  const dailyChart = data?.daily.slice(0, 14).map((day) => ({
    ...dailyChartValues(day),
    label: localDateLabel(day.date, false, locale),
  }));
  const activeSignals =
    view === "hourly" ? data?.signals : data?.outlookSignals;
  return (
    <ApplicationShell>
      <main>
        <header className="page-header">
          <div>
            <span className="eyebrow">{t("weather.page_eyebrow")}</span>
            <Typography component="h1" variant="h1">
              {t("weather.app_title")}
            </Typography>
            <p>{t("weather.page_description")}</p>
          </div>
          <FormControl className="city-control" size="small">
            <InputLabel
              id="forecast-location-label"
              className="city-control-label"
              shrink
              disableAnimation
            >
              {t("weather.forecast_location")}
            </InputLabel>
            <Select
              id="forecast-location"
              labelId="forecast-location-label"
              value={city}
              onChange={(e) => setCity(+e.target.value)}
            >
              {cities.map((c, i) => (
                <MenuItem value={i} key={c[0]}>
                  {c[0] === "Stockholm" ? t("weather.stockholm") : c[0]}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </header>
        {loading && <LoadingState />}
        {error && (
          <Alert severity="error" className="forecast-error">
            {t(error)}{" "}
            <Button onClick={() => setRetry((value) => value + 1)}>
              {t("weather.retry")}
            </Button>
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
            {data.hourly.length === 0 && data.daily.length === 0 && (
              <EmptyForecastState />
            )}
            {data.daily.length > 0 && <DailyCards data={data} />}
            <TradingInsights view={view} itemsForView={activeSignals} />
            {(view === "hourly"
              ? chart.length > 0
              : (dailyChart?.length ?? 0) > 0) && (
              <ForecastViews
                view={view}
                data={data}
                chart={chart}
                dailyChart={dailyChart}
              />
            )}
            <footer>
              <p>{t("weather.electricity_retail_weather_indicators")}</p>
              {t("weather.timezone")} {data.timezone} ·{" "}
              {t("weather.forecast_source")} Open-Meteo · {t("weather.updated")}{" "}
              {new Date(data.fetchedAt).toLocaleTimeString(locale)}
            </footer>
          </>
        )}
      </main>
    </ApplicationShell>
  );
}
