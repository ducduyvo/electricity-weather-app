import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { stripTypeScriptTypes } from "node:module";
import test from "node:test";
const source = stripTypeScriptTypes(
  readFileSync(
    new URL("../src/utils/forecastTime.ts", import.meta.url),
    "utf8",
  ),
).replace(/export /g, "");
const { currentForecastHour, hourlyPointLabel } = new Function(
  source + ";return {currentForecastHour,hourlyPointLabel};",
)();

test("DST fallback retains the current occurrence of the local hour", () => {
  const first = currentForecastHour(
    "Europe/Helsinki",
    new Date("2026-10-25T00:30:00Z"),
  );
  const second = currentForecastHour(
    "Europe/Helsinki",
    new Date("2026-10-25T01:30:00Z"),
  );
  assert.equal(second - first, 3600);
  assert.equal(first, Date.parse("2026-10-25T00:00:00Z") / 1000);
  assert.equal(second, Date.parse("2026-10-25T01:00:00Z") / 1000);
  const firstLabel = hourlyPointLabel(first, "Europe/Helsinki", "en", true);
  const secondLabel = hourlyPointLabel(second, "Europe/Helsinki", "en", true);
  assert.match(firstLabel, /03:00/);
  assert.match(secondLabel, /03:00/);
  assert.notEqual(firstLabel, secondLabel);
});

test("fractional-hour timezones use local hour boundaries", () => {
  assert.equal(
    currentForecastHour("Asia/Kathmandu", new Date("2026-10-13T06:05:00Z")),
    Date.parse("2026-10-13T05:15:00Z") / 1000,
  );
});
