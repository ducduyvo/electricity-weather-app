import Divider from "@mui/material/Divider";
import { formatMeasurement } from "../../utils/measurement";
import {
  dailyChartValues,
  dailySampleCount,
  type DailyMetric,
  isDailyMetricComplete,
} from "../../utils/dailyForecast";
import { WeatherMetricIcon } from "./WeatherMetricIcon";
import Typography from "@mui/material/Typography";
import { useI18n } from "../../i18n";
import type { Daily, Forecast } from "../../types/weather";
import { InfoTooltip } from "../common/InfoTooltip";
import { StatCard } from "../common/Surface";

const cardinalDirections = ["N", "NE", "E", "SE", "S", "SW", "W", "NW"];

function MetricCoverageNote({
  day,
  metric,
  value,
}: {
  day: Daily;
  metric: DailyMetric;
  value: number | null;
}) {
  const { t } = useI18n();
  const count = dailySampleCount(day, metric) ?? 0;
  const unavailable = count === 0 || value == null || !Number.isFinite(value);
  if (!unavailable && isDailyMetricComplete(day, metric)) return null;
  const unavailableKey =
    metric === "temperature"
      ? "weather.card_temperature_unavailable"
      : metric === "wind"
      ? "weather.card_wind_unavailable"
      : "weather.card_solar_unavailable";
  return (
    <small className="daily-metric-note">
      {!unavailable && (
        <svg
          className="daily-partial-warning-icon"
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          aria-hidden="true"
          focusable="false"
        >
          <path d="M12 3 2 21h20Z" strokeLinejoin="round" />
          <path d="M12 9v5m0 3v1" strokeLinecap="round" />
        </svg>
      )}
      {unavailable
        ? t(unavailableKey)
        : t(
            metric === "solar"
              ? "weather.card_incomplete_solar"
              : "weather.card_incomplete_average",
            { available: count, expected: day.expectedHourCount },
          )}
    </small>
  );
}

export function DailyCards({ data }: { data: Forecast }) {
  const { t, locale } = useI18n();
  return (
    <section aria-labelledby="daily-summary-heading">
      <Typography
        id="daily-summary-heading"
        component="h2"
        variant="h2"
        className="forecast-section-heading"
      >
        {t("weather.daily_summary")}
      </Typography>
      <div className="cards">
        {data.daily
          .slice(0, 3)
          .map(dailyChartValues)
          .map((d) => (
            <StatCard key={d.date}>
              <div className="daily-card-heading">
                <span>
                  {new Date(`${d.date}T12:00:00Z`).toLocaleDateString(locale, {
                    timeZone: "UTC",
                    weekday: "long",
                    month: "long",
                    day: "numeric",
                  })}
                </span>
              </div>
              <Divider className="daily-card-divider" />
              <small>{t("weather.daily_average_temperature")}</small>
              <strong
                className="daily-card-temperature"
                aria-label={t("weather.daily_average_temperature")}
              >
                {formatMeasurement(d.avgTemperature, "°C")}
              </strong>
              <small>
                {t("weather.daily_min")}{" "}
                {formatMeasurement(d.minTemperature, "°C")} {"·"}{" "}
                {t("weather.daily_max")}{" "}
                {formatMeasurement(d.maxTemperature, "°C")}
              </small>
              <MetricCoverageNote
                day={d}
                metric="temperature"
                value={d.avgTemperature}
              />
              <Divider className="daily-card-divider" />
              <small className="daily-solar-summary daily-card-row">
                <span className="daily-card-row-label">
                  <WeatherMetricIcon kind="solar" />
                  {t("weather.daily_solar")}
                </span>
                <span className="daily-card-row-value">
                  {formatMeasurement(
                    d.solarSampleCount > 0 ? d.solarEnergyKWhM2 : null,
                    " kWh/m²",
                    2,
                  )}
                </span>
              </small>
              <MetricCoverageNote
                day={d}
                metric="solar"
                value={d.solarSampleCount > 0 ? d.solarEnergyKWhM2 : null}
              />
              <Divider className="daily-card-divider" />
              <small className="daily-wind-summary daily-card-row">
                <span className="daily-card-row-label">
                  <WeatherMetricIcon kind="wind" />
                  <span className="daily-card-wind-label">
                    {t("weather.daily_wind_avg")}
                  </span>
                  <InfoTooltip text={t("weather.wind_explanation")} />
                </span>
                <span className="daily-card-row-value">
                  {formatMeasurement(d.avgWindSpeed, " m/s")}
                </span>
              </small>
              <MetricCoverageNote
                day={d}
                metric="wind"
                value={d.avgWindSpeed}
              />
              {d.avgWindDirection != null &&
                Number.isFinite(d.avgWindDirection) && (
                  <>
                    <Divider className="daily-card-divider" />
                    <small
                      className="daily-wind-direction daily-card-row"
                      title={t("weather.daily_wind_from", {
                        direction:
                          cardinalDirections[
                            Math.round(d.avgWindDirection / 45) % 8
                          ],
                      })}
                    >
                      {t("weather.prevailing_wind_direction", {
                        direction:
                          cardinalDirections[
                            Math.round(d.avgWindDirection / 45) % 8
                          ],
                        degrees: Math.round(d.avgWindDirection) % 360,
                      })
                        .split(":")
                        .map((part, index) =>
                          index === 0 ? (
                            <span key={index}>{part}</span>
                          ) : (
                            <span
                              key={index}
                              className="daily-card-row-value daily-card-direction-value"
                            >
                              {" "}
                              <svg
                                width="16"
                                height="16"
                                viewBox="0 0 16 16"
                                fill="none"
                                stroke="currentColor"
                                strokeWidth="1.5"
                                aria-hidden="true"
                                style={{
                                  transform: `rotate(${
                                    ((d.avgWindDirection ?? 0) + 180) % 360
                                  }deg)`,
                                }}
                              >
                                <path d="M8 14V2M3 7l5-5 5 5" />
                              </svg>
                              {part.trim()}
                            </span>
                          ),
                        )}
                    </small>
                  </>
                )}
            </StatCard>
          ))}
      </div>
    </section>
  );
}
