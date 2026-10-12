import { WeatherMetricIcon } from "./WeatherMetricIcon";
import Typography from "@mui/material/Typography";
import { useI18n } from "../../i18n";
import type { Forecast } from "../../types/weather";
import { InfoTooltip } from "../common/InfoTooltip";
import { StatCard } from "../common/Surface";

const cardinalDirections = ["N", "NE", "E", "SE", "S", "SW", "W", "NW"];

export function DailyCards({ data }: { data: Forecast }) {
  const { t, locale } = useI18n();
  return (
    <section aria-labelledby="daily-summary-heading">
      <Typography id="daily-summary-heading" component="h2" variant="h2" className="forecast-section-heading">{t("weather.daily_summary")}</Typography>
      <div className="cards">
      {data.daily.slice(0, 3).map((d) => (
        <StatCard key={d.date}>
          <div className="daily-card-heading">
            <span>{new Date(`${d.date}T12:00:00Z`).toLocaleDateString(locale, { timeZone: "UTC", weekday: "long", month: "long", day: "numeric" })}</span>
          </div>
          <small>{t("weather.daily_average_temperature")}</small>
          <strong className="daily-card-temperature" aria-label={t("weather.daily_average_temperature")}>{d.avgTemperature?.toFixed(1) ?? "—"}°C</strong>
          <small>
            {t("weather.daily_min")} {d.minTemperature?.toFixed(1) ?? "—"}°C · {t("weather.daily_max")} {d.maxTemperature?.toFixed(1) ?? "—"}°C
          </small>
          <small className="daily-solar-summary">
            <WeatherMetricIcon kind="solar" />
            {t("weather.daily_solar")}: {" "}
            {d.solarSampleCount > 0 ? d.solarEnergyKWhM2.toFixed(2) : "—"}{" "}
            kWh/m²
            {d.solarSampleCount > 0 &&
              (!d.completeDay || d.solarSampleCount < d.hourCount) &&
              t("weather.partial")}
          </small>
          <small className="daily-wind-summary">
            <WeatherMetricIcon kind="wind" />
            {t("weather.daily_wind_mean")}: {d.meanWindSpeed?.toFixed(1) ?? "—"} m/s
            {d.windSampleCount > 0 && (!d.completeDay || d.windSampleCount < d.hourCount) && t("weather.partial")}
            <InfoTooltip text={t("weather.wind_explanation")} />
          </small>
            {d.meanWindDirection != null && Number.isFinite(d.meanWindDirection) && (
              <small className="daily-wind-direction" title={t("weather.daily_wind_from", { direction: cardinalDirections[Math.round(d.meanWindDirection / 45) % 8] })}>
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true" style={{ transform: `rotate(${d.meanWindDirection}deg)` }}>
                  <path d="M8 14V2M3 7l5-5 5 5" />
                </svg>
                {t("weather.prevailing_wind_direction", { direction: cardinalDirections[Math.round(d.meanWindDirection / 45) % 8], degrees: Math.round(d.meanWindDirection) % 360 })}
              </small>
            )}
        </StatCard>
      ))}
    </div>
    </section>
  );
}
