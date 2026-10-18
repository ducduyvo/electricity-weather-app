import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { stripTypeScriptTypes } from "node:module";
import test from "node:test";

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
const source = stripTypeScriptTypes(
  readFileSync(new URL("../src/i18n/signals.ts", import.meta.url), "utf8"),
)
  .replace(/^import .*;\n/gm, "")
  .replace(/export /g, "");
const translate = new Function("english", source + ";return translateSignal;")({
  weather: en,
});
const translator =
  (locale) =>
  (key, values = {}) =>
    (locale === "fi" ? fi : en)[key.slice(8)].replace(
      /\{(\w+)\}/g,
      (_, name) => values[name],
    );
const wind = (code, change) => ({
  kind: "wind",
  severity: "info",
  title: "Changed backend title",
  detail: "Changed backend wording",
  code,
  params: { change, start: "2026-10-12T16:00", end: "2026-10-13T21:00" },
});

test("wind codes translate independently of fallback wording in both locales", () => {
  for (const locale of ["en", "fi"]) {
    for (const [code, change, titleKey, implicationKey] of [
      [
        "wind_speed_increasing",
        3.27,
        "wind_conditions_improving",
        "wind_increasing_implication",
      ],
      [
        "wind_speed_decreasing",
        -7.74,
        "wind_conditions_weakening",
        "wind_decreasing_implication",
      ],
    ]) {
      const t = translator(locale);
      const result = translate(wind(code, change), t, locale);
      assert.equal(result.title, t(`weather.${titleKey}`));
      assert.ok(result.detail.includes(t(`weather.${implicationKey}`)));
      assert.ok(
        result.detail.includes(
          new Intl.NumberFormat(locale, {
            minimumFractionDigits: 1,
            maximumFractionDigits: 1,
            signDisplay: "always",
          }).format(change),
        ),
      );
      assert.doesNotMatch(result.detail, /Changed backend|2026-10-/);
      for (const time of ["16:00", "21:00"])
        assert.match(result.detail, new RegExp(time.replace(":", "[.:]")));
    }
  }
});

test("daily wind codes format dates without inventing hourly times", () => {
  const signal = wind("wind_speed_increasing", 4);
  signal.params.start = "2026-10-12";
  signal.params.end = "2026-10-14";
  const result = translate(signal, translator("en"), "en");
  assert.match(result.detail, /12 Oct 2026/);
  assert.match(result.detail, /14 Oct 2026/);
  assert.doesNotMatch(result.detail, /12:00/);
});

test("unknown, legacy and invalid structured signals fall back safely", () => {
  for (const override of [
    { code: undefined },
    { code: "future_signal" },
    { params: {} },
    { params: { change: NaN, start: "2026-10-12", end: "2026-10-14" } },
    { params: { change: 4, start: "bad date", end: "2026-10-14" } },
    { params: { change: -4, start: "2026-10-12", end: "2026-10-14" } },
  ]) {
    const result = translate(
      { ...wind("wind_speed_increasing", 4), ...override },
      translator("fi"),
      "fi",
    );
    assert.deepEqual(result, {
      title: "Changed backend title",
      detail: "Changed backend wording",
    });
  }
  const result = translate(
    {
      kind: "temperature",
      severity: "warning",
      title: "Colder tomorrow",
      detail: "Unknown description",
    },
    translator("fi"),
    "fi",
  );
  assert.equal(result.title, fi.colder_tomorrow);
  assert.equal(result.detail, "Unknown description");
});

test("temperature and solar codes render both locales without reading fallback sentences", () => {
  const cases = [
    [
      "temperature_decreasing",
      { change: -4, interpretation: "heating_increase" },
      "colder_tomorrow",
    ],
    [
      "temperature_increasing",
      { change: 4, interpretation: "cooling_increase" },
      "warmer_tomorrow",
    ],
    [
      "temperature_trend",
      {
        change: -4,
        start: "2026-10-12",
        end: "2026-10-15",
        trend: "cold_spell",
        interpretation: "heating_increase",
      },
      "cold_spell_developing",
    ],
    [
      "temperature_trend",
      {
        change: -4,
        start: "2026-10-12",
        end: "2026-10-15",
        trend: "cooling",
        interpretation: "cooling_ease",
      },
      "cooling_trend",
    ],
    [
      "temperature_trend",
      {
        change: 4,
        start: "2026-10-12",
        end: "2026-10-15",
        trend: "warming",
        interpretation: "heating_ease",
      },
      "warming_trend",
    ],
    [
      "solar_daily_decreasing",
      { change: -1.5, date: "2026-10-13", baseline: "2026-10-12" },
      "lower_solar_availability",
    ],
    [
      "solar_daily_increasing",
      { change: 1.5, date: "2026-10-13", baseline: "2026-10-12" },
      "higher_solar_availability",
    ],
    [
      "solar_trend_decreasing",
      { change: -1.5, start: "2026-10-12", end: "2026-10-15" },
      "solar_availability_weakening",
    ],
    [
      "solar_trend_increasing",
      { change: 1.5, start: "2026-10-12", end: "2026-10-15" },
      "solar_availability_improving",
    ],
    [
      "solar_peak",
      { date: "2026-10-12", time: "13:00", peak: 400, partial: false },
      "solar_period_expected",
    ],
    [
      "solar_peak_strong",
      { date: "2026-10-12", time: "13:00", peak: 600, partial: true },
      "strong_solar_period_expected",
    ],
  ];
  for (const locale of ["en", "fi"])
    for (const [code, params, titleKey] of cases) {
      const t = translator(locale);
      const result = translate(
        {
          kind: "weather",
          severity: "info",
          code,
          params,
          title: "Different wording",
          detail: "Different description",
        },
        t,
        locale,
      );
      assert.equal(result.title, t(`weather.${titleKey}`));
      assert.doesNotMatch(result.detail, /Different|undefined|2026-10-/);
      if (params.time) assert.ok(result.detail.includes("13:00"));
      if (params.partial)
        assert.ok(
          result.detail.endsWith(
            t("weather.some_hourly_readings_are_missing_this_is"),
          ),
        );
      if (params.change && code.startsWith("solar"))
        assert.ok(
          result.detail.includes(
            new Intl.NumberFormat(locale, {
              minimumFractionDigits: 2,
              maximumFractionDigits: 2,
              signDisplay: code.startsWith("solar_trend") ? "always" : "auto",
            }).format(
              code.startsWith("solar_trend")
                ? params.change
                : Math.abs(params.change),
            ),
          ),
        );
    }
});

test("all short-term temperature interpretations are preserved", () => {
  const keys = {
    heating_increase:
      "colder_conditions_may_increase_heating_driven_electricity",
    cooling_ease: "cooling_driven_electricity_demand_conditions_may_ease",
    cooling_increase:
      "warmer_conditions_may_increase_cooling_driven_electricity",
    heating_ease: "heating_driven_electricity_demand_conditions_may_ease",
    dependent: "the_electricity_demand_implication_depends_on_heating",
  };
  for (const [interpretation, key] of Object.entries(keys)) {
    const result = translate(
      {
        kind: "temperature",
        severity: "info",
        code: "temperature_increasing",
        params: { change: 3, interpretation },
        title: "Changed",
        detail: "Changed",
      },
      translator("fi"),
      "fi",
    );
    assert.ok(result.detail.endsWith(fi[key]));
  }
});

test("malformed temperature and solar metadata falls back without crashing", () => {
  for (const [code, params] of [
    [
      "temperature_trend",
      {
        change: 4,
        start: "2026-10-12",
        end: "2026-10-15",
        trend: "__proto__",
        interpretation: "dependent",
      },
    ],
    ["temperature_increasing", { change: 3, interpretation: "unknown" }],
    [
      "solar_peak",
      { date: "2026-10-12", time: "25:00", peak: 400, partial: false },
    ],
    [
      "solar_daily_decreasing",
      { change: 2, date: "2026-10-12", baseline: "2026-10-11" },
    ],
  ]) {
    assert.deepEqual(
      translate(
        {
          kind: "weather",
          severity: "info",
          title: "Original",
          detail: "Original detail",
          code,
          params,
        },
        translator("fi"),
        "fi",
      ),
      { title: "Original", detail: "Original detail" },
    );
  }
});
