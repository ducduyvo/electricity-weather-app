import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { mkdtemp, rm } from "node:fs/promises";
import { stripTypeScriptTypes, createRequire } from "node:module";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { build } from "rolldown";

const source = stripTypeScriptTypes(
  readFileSync(
    new URL("../src/utils/dailyForecast.ts", import.meta.url),
    "utf8",
  ),
)
  .replace(/^import .*;\n/gm, "")
  .replace(/export /g, "");
const { dailyChartValues, isDailyMetricComplete } = new Function(
  source + ";return {dailyChartValues, isDailyMetricComplete};",
)();
const completeDay = (hours = 24) => ({
  completeDay: true,
  hourCount: hours,
  expectedHourCount: hours,
  temperatureSampleCount: hours,
  windSampleCount: hours,
  solarSampleCount: hours,
  avgTemperature: 5.087500000000001,
  minTemperature: 2.6,
  maxTemperature: 8.9,
  avgWindSpeed: 3.4,
  minWindSpeed: 0.9,
  maxWindSpeed: 6,
  solarEnergyKWhM2: 1.4,
});

test("partial chart styling uses metric counts and DST-aware expected coverage", () => {
  for (const hours of [23, 24, 25]) {
    const day = completeDay(hours);
    assert.equal(isDailyMetricComplete(day, "wind"), true);
    assert.equal(
      isDailyMetricComplete({ ...day, windSampleCount: hours - 1 }, "wind"),
      false,
    );
    assert.equal(
      isDailyMetricComplete(
        { ...day, windSampleCount: hours - 1 },
        "temperature",
      ),
      true,
    );
    assert.equal(
      isDailyMetricComplete({ ...day, completeDay: false }, "temperature"),
      false,
    );
    assert.equal(
      isDailyMetricComplete({ ...day, solarSampleCount: 0 }, "solar"),
      false,
    );
  }
});

test("backend completeness flags take precedence with legacy fallback", () => {
  for (const [metric, field] of [
    ["temperature", "completeTemperature"],
    ["wind", "completeWind"],
    ["solar", "completeSolar"],
  ]) {
    assert.equal(
      isDailyMetricComplete({ ...completeDay(), [field]: false }, metric),
      false,
    );
    assert.equal(
      isDailyMetricComplete(
        { ...completeDay(), completeDay: false, [field]: true },
        metric,
      ),
      true,
    );
    assert.equal(isDailyMetricComplete(completeDay(25), metric), true);
  }
  assert.equal(
    dailyChartValues({ ...completeDay(), completeSolar: false }).solarEnergy,
    null,
  );
  assert.equal(
    dailyChartValues({
      ...completeDay(),
      completeDay: false,
      completeSolar: true,
    }).solarEnergy,
    1.4,
  );
});

test("chart values exclude zero-sample measurements without inventing zeros", () => {
  const result = dailyChartValues({
    ...completeDay(),
    temperatureSampleCount: 0,
    windSampleCount: 0,
    solarSampleCount: 0,
    avgTemperature: 0,
    avgWindSpeed: 0,
    solarEnergyKWhM2: 0,
  });
  for (const field of [
    "avgTemperature",
    "minTemperature",
    "maxTemperature",
    "avgWindSpeed",
    "minWindSpeed",
    "maxWindSpeed",
    "solarEnergy",
  ])
    assert.equal(result[field], null);
  const zero = dailyChartValues({
    ...completeDay(),
    avgTemperature: 0,
    avgWindSpeed: 0,
    solarEnergyKWhM2: 0,
  });
  assert.equal(zero.avgTemperature, 0);
  assert.equal(zero.avgWindSpeed, 0);
  assert.equal(zero.solarEnergy, 0);
});

test("partial averages stay available while incomplete accumulated solar stays a gap", () => {
  const input = {
    ...completeDay(25),
    completeDay: false,
    hourCount: 20,
    temperatureSampleCount: 18,
    windSampleCount: 20,
    solarSampleCount: 20,
  };
  const result = dailyChartValues(input);
  assert.equal(result.avgTemperature, input.avgTemperature);
  assert.equal(result.avgWindSpeed, input.avgWindSpeed);
  assert.equal(result.solarEnergy, null);
  assert.equal(input.solarEnergyKWhM2, 1.4);
});

const output = await mkdtemp(join(tmpdir(), "weather-coverage-"));
try {
  await build({
    input: new URL("./fixtures/coverage-tooltip.tsx", import.meta.url).pathname,
    platform: "node",
    transform: { jsx: { runtime: "automatic" } },
    onwarn(warning, handler) {
      if (warning.code !== "MODULE_LEVEL_DIRECTIVE") handler(warning);
    },
    output: { file: join(output, "tooltip.cjs"), format: "cjs" },
  });
  const {
    renderCoverage,
    renderPartialLine,
    renderDailyCards,
    renderDailyTable,
  } = createRequire(import.meta.url)(join(output, "tooltip.cjs"));
  test("daily cards distinguish missing values, partial measurements, and valid zeros", () => {
    const missing = renderDailyCards({
      ...completeDay(),
      temperatureSampleCount: 0,
      windSampleCount: 0,
      solarSampleCount: 0,
      avgTemperature: 99,
      avgWindSpeed: 99,
    });
    assert.match(missing, /—/);
    assert.doesNotMatch(missing, /99.0/);
    assert.doesNotMatch(missing, /\(partial\)/);
    const partial = renderDailyCards({
      ...completeDay(),
      completeTemperature: false,
      completeWind: false,
      completeSolar: false,
    });
    assert.equal((partial.match(/Incomplete data/g) ?? []).length, 3);
    const zero = renderDailyCards({
      ...completeDay(),
      avgTemperature: 0,
      avgWindSpeed: 0,
      solarEnergyKWhM2: 0,
    });
    assert.match(zero, /0.0/);
    assert.match(zero, /0.00/);
    assert.doesNotMatch(zero, /—/);
  });
  test("daily card explanations report actual metric counts and suppress notes for complete zeros", () => {
    const html = renderDailyCards({
      ...completeDay(),
      completeTemperature: false,
      completeWind: false,
      completeSolar: false,
      temperatureSampleCount: 22,
      windSampleCount: 20,
      solarSampleCount: 18,
    });
    assert.equal((html.match(/Incomplete data/g) ?? []).length, 3);
    assert.equal((html.match(/daily-partial-warning-icon/g) ?? []).length, 3);
    assert.match(html, /22\/24/);
    assert.match(html, /20\/24/);
    assert.match(html, /18\/24/);
    assert.match(html, /Average based on available readings/);
    assert.match(html, /Daily energy total may be underestimated/);
    const missing = renderDailyCards({
      ...completeDay(),
      temperatureSampleCount: 0,
      windSampleCount: 0,
      solarSampleCount: 0,
    });
    for (const metric of ["Temperature", "Wind", "Solar"])
      assert.ok(missing.includes(`${metric} data unavailable.`));
    assert.doesNotMatch(missing, /Incomplete data|—°C|— m\/s|— kWh\/m²/);
    const zero = renderDailyCards({
      ...completeDay(),
      avgTemperature: 0,
      avgWindSpeed: 0,
      solarEnergyKWhM2: 0,
    });
    assert.doesNotMatch(zero, /Partial data|Incomplete data|data unavailable/);
    assert.match(zero, /0.0°C/);
    assert.match(zero, /0.0 m\/s/);
    assert.match(zero, /0.00 kWh\/m²/);
  });
  test("daily card layout separates sections and aligns metric values", () => {
    const html = renderDailyCards({ ...completeDay(), avgWindDirection: 180 });
    assert.equal((html.match(/<hr/g) ?? []).length, 4);
    assert.match(html, /daily-card-row-label/);
    assert.match(html, /daily-card-direction-value/);
    assert.match(html, /Prevailing/);
    assert.match(html, /S \(180°\)/);
    assert.match(html, /3.4 m\/s/);
  });
  test("daily table follows backend completeness flags and sanitizes missing values", () => {
    const partial = renderDailyTable({
      ...completeDay(),
      completeTemperature: false,
      completeWind: false,
      completeSolar: false,
    });
    assert.equal((partial.match(/\(partial\)/g) ?? []).length, 3);
    const missing = renderDailyTable({
      ...completeDay(),
      temperatureSampleCount: 0,
      windSampleCount: 0,
      solarSampleCount: 0,
      avgTemperature: 99,
      avgWindSpeed: 99,
      peakSolarIrradiance: 999,
    });
    assert.doesNotMatch(missing, /99.0|999 \/|NaN/);
    const zero = renderDailyTable({
      ...completeDay(),
      avgTemperature: 0,
      avgWindSpeed: 0,
      solarEnergyKWhM2: 0,
    });
    assert.match(zero, /0.00/);
    assert.doesNotMatch(zero, /\(partial\)/);
  });
  test("only incoming partial segments are dashed and missing values remain gaps", () => {
    const full = completeDay();
    const partial = { ...full, windSampleCount: 23 };
    const html = renderPartialLine(
      [full, partial, full, full, full],
      [5, 4, 3, null, 2],
    );
    assert.equal((html.match(/<path/g) ?? []).length, 2);
    assert.equal((html.match(/stroke-dasharray="5 5"/g) ?? []).length, 1);
    assert.equal((html.match(/fill="var\(--surface\)"/g) ?? []).length, 1);
    assert.equal((html.match(/<circle/g) ?? []).length, 4);
    assert.match(html, /stroke-width="3"/);
  });
  test("complete to partial to complete transitions retain segment order for both metrics", () => {
    for (const metric of ["temperature", "wind"]) {
      const full = completeDay(25);
      const countKey =
        metric === "temperature" ? "temperatureSampleCount" : "windSampleCount";
      const partial = { ...full, [countKey]: 24 };
      const html = renderPartialLine(
        [full, partial, partial, full],
        [0, 4, 3, 2],
        metric,
      );
      const paths = html.match(/<path[^>]*>/g) ?? [];
      assert.equal(paths.length, 3);
      assert.ok(
        paths.every((path) => path.includes(`stroke="var(--chart-${metric})"`)),
      );
      assert.equal(
        (html.match(new RegExp(`fill="var\\(--chart-${metric}\\)"`, "g")) ?? [])
          .length,
        2,
      );
      assert.match(paths[0], /stroke-dasharray="5 5"/);
      assert.match(paths[1], /stroke-dasharray="5 5"/);
      assert.doesNotMatch(paths[2], /stroke-dasharray/);
      assert.equal((html.match(/fill="var\(--surface\)"/g) ?? []).length, 2);
      assert.equal((html.match(/<circle/g) ?? []).length, 4);
      const other = metric === "temperature" ? "wind" : "temperature";
      assert.doesNotMatch(
        renderPartialLine([full, partial, full], [0, 4, 2], other),
        /stroke-dasharray/,
      );
    }
  });
  test("missing endpoints and consecutive missing points do not bridge gaps", () => {
    const full = completeDay();
    for (const metric of ["temperature", "wind"]) {
      for (const values of [[], [null], [null, 3, null], [5, null, null, 2]]) {
        const html = renderPartialLine(
          values.map(() => full),
          values,
          metric,
        );
        assert.doesNotMatch(html, /<path/);
        assert.equal(
          (html.match(/<circle/g) ?? []).length,
          values.filter((value) => value != null).length,
        );
      }
    }
  });
  test("complete 23-hour and 25-hour days are not labeled partial", () => {
    for (const hours of [23, 25])
      for (const metric of ["temperature", "wind", "solar"]) {
        assert.doesNotMatch(
          renderCoverage(metric, completeDay(hours)),
          /Partial data/,
        );
      }
  });
  test("tooltip distinguishes day coverage and metric-specific missing readings", () => {
    const day = {
      ...completeDay(25),
      completeDay: false,
      hourCount: 20,
      temperatureSampleCount: 18,
      windSampleCount: 19,
      solarSampleCount: 20,
    };
    for (const [metric, count] of [
      ["temperature", 18],
      ["wind", 19],
      ["solar", 20],
    ]) {
      const html = renderCoverage(metric, day);
      assert.match(html, /Partial data/);
      assert.ok(
        html.includes(`Available measurements: ${count} / 25 expected hours`),
      );
      assert.match(html, /Hourly coverage: 20 \/ 25 expected hours/);
    }
    const missing = { ...completeDay(), windSampleCount: 22 };
    const html = renderCoverage("wind", missing);
    assert.match(html, /Available measurements: 22 \/ 24 expected hours/);
    assert.match(html, /Some recorded hours have no valid readings/);
    assert.doesNotMatch(html, /Hourly coverage:/);
    assert.doesNotMatch(renderCoverage("temperature", missing), /Partial data/);
  });
  test("tooltip keeps one decimal and shows no-sample values as missing", () => {
    assert.match(
      renderCoverage("temperature", completeDay()),
      /Average: <span class="tooltip-metric-value">5.1 °C<\/span>/,
    );
    const html = renderCoverage("solar", {
      ...completeDay(),
      solarSampleCount: 0,
      solarEnergyKWhM2: 0,
    });
    assert.match(html, /Available measurements: 0 \/ 24 expected hours/);
    assert.match(
      html,
      /Solar energy: <span class="tooltip-metric-value">—<\/span>/,
    );
    assert.doesNotMatch(html, /Solar energy: 0.0/);
  });
  test("coverage translations have matching placeholders in English and Finnish", () => {
    const en = JSON.parse(
      readFileSync(
        new URL("../src/i18n/locales/en/weather.json", import.meta.url),
        "utf8",
      ),
    ).weather;
    const fi = JSON.parse(
      readFileSync(
        new URL("../src/i18n/locales/fi/weather.json", import.meta.url),
        "utf8",
      ),
    ).weather;
    for (const key of [
      "chart_complete_day",
      "chart_partial_day",
      "tooltip_partial_data",
      "tooltip_measurement_coverage",
      "tooltip_hourly_coverage",
      "tooltip_missing_measurements",
    ]) {
      assert.ok(en[key] && fi[key]);
      assert.deepEqual(
        (en[key].match(/\{\w+\}/g) ?? []).sort(),
        (fi[key].match(/\{\w+\}/g) ?? []).sort(),
      );
    }
  });
} finally {
  await rm(output, { recursive: true, force: true });
}
