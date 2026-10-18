import { chartMetricColors } from "../../designTokens";
import { useI18n, type TranslationKey } from "../../i18n";
import type { Forecast, WindPotential } from "../../types/weather";

const potentialKeys: Record<WindPotential, TranslationKey> = {
  very_low: "weather.wind_very_low",
  low: "weather.wind_low",
  moderate: "weather.wind_moderate",
  high: "weather.wind_high",
  strong_uncertain: "weather.wind_strong_uncertain",
};

type WindTooltipProps = {
  active?: boolean;
  label?: string | number;
  payload?: readonly { payload?: Forecast["hourly"][number] }[];
};

export function WindTooltip({ active, label, payload }: WindTooltipProps) {
  const { t, locale } = useI18n();
  const point = payload?.[0]?.payload;
  if (!active || !point) return null;
  return (
    <div className="wind-chart-tooltip">
      <p>
        <b>{label}</b>
      </p>
      <p>
        <span
          className="tooltip-metric-indicator"
          style={{ backgroundColor: chartMetricColors.wind }}
          aria-hidden="true"
        />
        {t("weather.wind_speed")}:{" "}
        <span className="tooltip-metric-value">
          {point.windSpeed != null && Number.isFinite(point.windSpeed)
            ? new Intl.NumberFormat(locale, {
                minimumFractionDigits: 1,
                maximumFractionDigits: 1,
              }).format(point.windSpeed)
            : "—"}
          {point.windSpeed != null && Number.isFinite(point.windSpeed)
            ? " m/s"
            : ""}
        </span>
      </p>
      <p>
        {point.windDirection != null && Number.isFinite(point.windDirection)
          ? t("weather.wind_direction", {
              cardinal: ["N", "NE", "E", "SE", "S", "SW", "W", "NW"][
                Math.round((((point.windDirection % 360) + 360) % 360) / 45) % 8
              ],
              direction: point.windDirection.toFixed(0),
            })
              .split(/(\d+(?:[.,]\d+)?°)/)
              .map((part, index) =>
                /\d/.test(part) ? (
                  <span key={index} className="tooltip-metric-value">
                    {part}
                  </span>
                ) : (
                  part
                ),
              )
          : t("weather.wind_direction_missing")}
      </p>
      {point.windPotential && (
        <p>
          {t("weather.wind_potential_label")}:{" "}
          {t(potentialKeys[point.windPotential])}
        </p>
      )}
    </div>
  );
}
