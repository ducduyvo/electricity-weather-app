import type { Daily } from "../types/weather";

export type DailyMetric = "temperature" | "wind" | "solar";

export function dailySampleCount(
  day: Partial<Daily>,
  metric: DailyMetric,
): number | undefined {
  return metric === "temperature"
    ? day.temperatureSampleCount
    : metric === "wind"
    ? day.windSampleCount
    : day.solarSampleCount;
}

export function isDailyMetricComplete(
  day: Partial<Daily>,
  metric: DailyMetric,
): boolean {
  const complete = metric === "temperature"
    ? day.completeTemperature
    : metric === "wind"
    ? day.completeWind
    : day.completeSolar;
  if (typeof complete === "boolean") return complete;
  return (
    day.completeDay === true &&
    typeof day.expectedHourCount === "number" &&
    day.expectedHourCount > 0 &&
    day.hourCount === day.expectedHourCount &&
    dailySampleCount(day, metric) === day.expectedHourCount
  );
}

function availableValue(value: number | null, count: number): number | null {
  return count > 0 && value != null && Number.isFinite(value) ? value : null;
}

export function dailyChartValues(day: Daily) {
  return {
    ...day,
    minTemperature: availableValue(
      day.minTemperature,
      day.temperatureSampleCount,
    ),
    maxTemperature: availableValue(
      day.maxTemperature,
      day.temperatureSampleCount,
    ),
    avgTemperature: availableValue(
      day.avgTemperature,
      day.temperatureSampleCount,
    ),
    minWindSpeed: availableValue(day.minWindSpeed, day.windSampleCount),
    maxWindSpeed: availableValue(day.maxWindSpeed, day.windSampleCount),
    avgWindSpeed: availableValue(day.avgWindSpeed, day.windSampleCount),
    // A partial accumulated total is not comparable to a full-day solar total.
    solarEnergy:
      isDailyMetricComplete(day, "solar")
        ? availableValue(day.solarEnergyKWhM2, day.solarSampleCount)
        : null,
  };
}
