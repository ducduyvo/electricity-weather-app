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
  const { t } = useI18n();
  const point = payload?.[0]?.payload;
  if (!active || !point) return null;
  return (
    <div className="wind-chart-tooltip">
      <p><b>{label}</b></p>
      <p>{t("weather.wind_speed")}: {point.windSpeed?.toFixed(1) ?? "—"} m/s</p>
      <p>{point.windDirection != null ? t("weather.wind_direction", { direction: point.windDirection.toFixed(0) }) : t("weather.wind_direction_missing")}</p>
      {point.windPotential && <p>{t("weather.wind_potential_label")}: {t(potentialKeys[point.windPotential])}</p>}
    </div>
  );
}
