import { chartMetricColors } from "../../designTokens";
import { Curve, type CurveProps } from "recharts";
import type { Daily } from "../../types/weather";
import {
  isDailyMetricComplete,
  type DailyMetric,
} from "../../utils/dailyForecast";

// Keep a single Recharts series for its existing legend and tooltip payload.
export function DailyForecastLine({
  days,
  metric,
  points = [],
  ...props
}: Partial<CurveProps> & { days: Daily[]; metric: DailyMetric }) {
  return (
    <g>
      {points.slice(1).map((current, index) => {
        const previous = points[index];
        if (
          previous.x == null ||
          previous.y == null ||
          current.x == null ||
          current.y == null
        )
          return null;

        return (
          <Curve
            {...props}
            key={index}
            pathRef={undefined}
            points={[previous, current]}
            strokeDasharray={
              isDailyMetricComplete(days[index + 1], metric) ? undefined : "5 5"
            }
          />
        );
      })}
    </g>
  );
}

export function DailyForecastDot({
  cx,
  cy,
  r,
  payload,
  metric,
}: {
  cx?: number;
  cy?: number;
  r: number;
  payload?: Daily;
  metric: DailyMetric;
}) {
  if (cx == null || cy == null || !payload) return null;
  return (
    <circle
      cx={cx}
      cy={cy}
      r={r}
      fill={
        isDailyMetricComplete(payload, metric)
          ? chartMetricColors[metric]
          : "var(--surface)"
      }
      stroke={chartMetricColors[metric]}
      strokeWidth={r === 4 ? 2 : 1}
    />
  );
}
