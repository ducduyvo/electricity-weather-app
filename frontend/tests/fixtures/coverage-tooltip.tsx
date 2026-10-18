import { ForecastViews } from "../../src/components/weather/ForecastViews";
import { DailyCards } from "../../src/components/weather/DailyCards";
import { chartMetricColors } from "../../src/designTokens";
import { renderToStaticMarkup } from "react-dom/server";
import { I18nProvider } from "../../src/i18n";
import { ForecastTooltip } from "../../src/components/weather/ForecastTooltip";
import {
  DailyForecastLine,
  DailyForecastDot,
} from "../../src/components/weather/DailyForecastLine";
import type { Daily } from "../../src/types/weather";

export function renderCoverage(
  metric: "temperature" | "wind" | "solar",
  day: Daily,
) {
  const dataKey =
    metric === "temperature"
      ? "avgTemperature"
      : metric === "wind"
      ? "avgWindSpeed"
      : "solarEnergy";
  const value =
    metric === "temperature"
      ? day.avgTemperature
      : metric === "wind"
      ? day.avgWindSpeed
      : day.solarEnergyKWhM2;
  return renderToStaticMarkup(
    <I18nProvider>
      <ForecastTooltip
        active
        daily
        metric={metric}
        label="Sun, Oct 25"
        payload={[{ dataKey, value, payload: day }]}
      />
    </I18nProvider>,
  );
}

export function renderPartialLine(
  days: Daily[],
  values: (number | null)[],
  metric: "temperature" | "wind" = "wind",
) {
  return renderToStaticMarkup(
    <svg>
      <DailyForecastLine
        days={days}
        metric={metric}
        stroke={chartMetricColors[metric]}
        strokeWidth={3}
        points={values.map((y, x) => ({ x, y }))}
      />
      {values.map((value, index) =>
        value == null ? null : (
          <DailyForecastDot
            key={index}
            cx={index}
            cy={value}
            r={3}
            payload={days[index]}
            metric={metric}
          />
        ),
      )}
    </svg>,
  );
}

export function renderDailyCards(day: Daily) {
  return renderToStaticMarkup(
    <I18nProvider>
      <DailyCards
        data={{
          timezone: "Europe/Helsinki",
          hourly: [],
          daily: [{ ...day, date: "2026-10-14" }],
          signals: [],
          outlookSignals: [],
          fetchedAt: "2026-10-14T00:00:00Z",
        }}
      />
    </I18nProvider>,
  );
}

export function renderDailyTable(day: Daily) {
  const data = {
    timezone: "Europe/Helsinki",
    hourly: [],
    daily: [{ ...day, date: "2026-10-14" }],
    signals: [],
    outlookSignals: [],
    fetchedAt: "2026-10-14T00:00:00Z",
  };
  return renderToStaticMarkup(
    <I18nProvider>
      <ForecastViews view="daily" data={data} chart={[]} dailyChart={[]} />
    </I18nProvider>,
  );
}
