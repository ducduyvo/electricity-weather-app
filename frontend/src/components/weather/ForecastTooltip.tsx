import { chartMetricColors } from "../../designTokens";
import { useI18n, type TranslationKey } from "../../i18n";
import {
  dailySampleCount,
  isDailyMetricComplete,
} from "../../utils/dailyForecast";
import type { Daily } from "../../types/weather";

type Metric = "temperature" | "wind" | "solar";
type ForecastTooltipProps = {
  active?: boolean;
  label?: string | number;
  payload?: readonly {
    dataKey?: string | number;
    value?: unknown;
    color?: string;
    payload?: Partial<Daily>;
  }[];
  metric: Metric;
  daily?: boolean;
};

const metricOrder = [
  "minTemperature",
  "minWindSpeed",
  "maxTemperature",
  "maxWindSpeed",
  "avgTemperature",
  "avgWindSpeed",
];

const labels: Record<string, TranslationKey> = {
  minTemperature: "weather.chart_min",
  avgTemperature: "weather.chart_average",
  maxTemperature: "weather.chart_max",
  minWindSpeed: "weather.chart_min",
  avgWindSpeed: "weather.chart_average",
  maxWindSpeed: "weather.chart_max",
  temperature: "weather.chart_temperature",
  solarIrradiance: "weather.chart_solar_irradiance",
  solarEnergy: "weather.chart_solar_energy",
};

export function ForecastTooltip({
  active,
  label,
  payload,
  metric,
  daily = false,
}: ForecastTooltipProps) {
  const { t, locale } = useI18n();
  if (!active || !payload?.length) return null;
  const day = payload[0].payload;
  const sampleCount = day ? dailySampleCount(day, metric) : undefined;
  const knownCount = (value: unknown): value is number =>
    typeof value === "number" && Number.isFinite(value) && value >= 0;
  const hasMeasurementCoverage =
    knownCount(sampleCount) && knownCount(day?.expectedHourCount);
  const hasHourlyCoverage =
    knownCount(day?.hourCount) && knownCount(day?.expectedHourCount);
  const incompleteHours =
    day?.completeDay === false ||
    (hasHourlyCoverage && day!.hourCount! < day!.expectedHourCount!);
  const missingMeasurements =
    knownCount(sampleCount) &&
    knownCount(day?.hourCount) &&
    sampleCount < day!.hourCount!;
  const partial =
    daily &&
    (incompleteHours ||
      missingMeasurements ||
      (day != null &&
        hasMeasurementCoverage &&
        !isDailyMetricComplete(day, metric)));
  const unit =
    metric === "temperature"
      ? "°C"
      : metric === "wind"
      ? "m/s"
      : daily
      ? "kWh/m²"
      : "W/m²";
  const format = new Intl.NumberFormat(locale, {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  });
  const entries =
    daily && day && (metric === "temperature" || metric === "wind")
      ? metric === "temperature"
        ? [
            {
              dataKey: "maxTemperature",
              value: day.maxTemperature,
              color: "var(--text-secondary)",
            },
            {
              dataKey: "avgTemperature",
              value: day.avgTemperature,
              color: chartMetricColors.temperature,
            },
            {
              dataKey: "minTemperature",
              value: day.minTemperature,
              color: "var(--text-secondary)",
            },
          ]
        : [
            {
              dataKey: "maxWindSpeed",
              value: day.maxWindSpeed,
              color: "var(--text-secondary)",
            },
            {
              dataKey: "avgWindSpeed",
              value: day.avgWindSpeed,
              color: chartMetricColors.wind,
            },
            {
              dataKey: "minWindSpeed",
              value: day.minWindSpeed,
              color: "var(--text-secondary)",
            },
          ]
      : payload;
  return (
    <div className="wind-chart-tooltip">
      <p>
        <b>{label}</b>
      </p>
      {partial && (
        <div className="forecast-tooltip-coverage">
          <span className="forecast-tooltip-partial">
            {t("weather.tooltip_partial_data")}
          </span>
          {hasMeasurementCoverage && (
            <p>
              {t("weather.tooltip_measurement_coverage", {
                available: sampleCount!,
                expected: day!.expectedHourCount!,
              })}
            </p>
          )}
          {incompleteHours && hasHourlyCoverage && (
            <p>
              {t("weather.tooltip_hourly_coverage", {
                available: day!.hourCount!,
                expected: day!.expectedHourCount!,
              })}
            </p>
          )}
          {missingMeasurements && (
            <p>{t("weather.tooltip_missing_measurements")}</p>
          )}
        </div>
      )}
      {[...entries]
        .sort(
          (a, b) =>
            metricOrder.indexOf(String(a.dataKey)) -
            metricOrder.indexOf(String(b.dataKey)),
        )
        .map((entry, index) => {
          const key = String(entry.dataKey);
          if (!Object.prototype.hasOwnProperty.call(labels, key)) return null;
          const value =
            (!daily || (knownCount(sampleCount) && sampleCount > 0)) &&
            typeof entry.value === "number" &&
            Number.isFinite(entry.value)
              ? format.format(entry.value)
              : "—";
          return (
            <p key={`${key}-${index}`}>
              <span
                className="tooltip-metric-indicator"
                style={{ backgroundColor: chartMetricColors[metric] }}
                aria-hidden="true"
              />
              {t(labels[key])}:{" "}
              <span className="tooltip-metric-value">
                {value}
                {value === "—" ? "" : ` ${unit}`}
              </span>
            </p>
          );
        })}
    </div>
  );
}
