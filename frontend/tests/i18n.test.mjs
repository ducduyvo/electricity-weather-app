import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { stripTypeScriptTypes } from "node:module";
import test from "node:test";

const load = (path) => stripTypeScriptTypes(readFileSync(new URL(path, import.meta.url), "utf8")).replace(/^import .*;\n/gm, "").replace(/export /g, "");
const en = JSON.parse(readFileSync(new URL("../src/i18n/locales/en/weather.json", import.meta.url), "utf8")).weather;
const fi = JSON.parse(readFileSync(new URL("../src/i18n/locales/fi/weather.json", import.meta.url), "utf8")).weather;
const translate = new Function("english", load("../src/i18n/signals.ts") + ";return translateSignalText;")({ weather: en });
const t = (key, values = {}) => fi[key.replace(/^weather\./, "")].replace(/\{(\w+)\}/g, (_, name) => values[name]);

test("English and Finnish have matching, nonempty entries and placeholders", () => {
  assert.deepEqual(Object.keys(fi).sort(), Object.keys(en).sort());
  for (const key of Object.keys(en)) {
    assert.ok(fi[key].length);
    assert.deepEqual((fi[key].match(/\{\w+\}/g) ?? []).sort(), (en[key].match(/\{\w+\}/g) ?? []).sort());
  }
});
test("dynamic insights preserve values and translate their interpretation", () => {
  const inputs = [
    "Average temperature is expected to fall 3.0°C tomorrow. Colder conditions may increase heating-driven electricity demand.",
    "Solar-energy proxy on 2026-10-11 rises 0.50 kWh/m² versus 2026-10-10. Higher irradiance may improve conditions for solar PV generation.",
    "Daily average temperature changes by -3.0°C from 2026-10-10 through 2026-10-14 across consecutive forecast days. Colder conditions may increase heating-driven electricity demand.",
    "The daily solar-energy proxy changes by +0.50 kWh/m² from 2026-10-10 through 2026-10-14 across consecutive forecast days. Higher irradiance may improve conditions for solar PV generation.",
    "Solar irradiance on 2026-10-11 is forecast to peak around 12:00 at 600 W/m², indicating favorable irradiance conditions for solar PV generation. Time is local to the forecast location. Some hourly readings are missing; this is the peak among available readings.",
  ];
  for (const input of inputs) {
    const translated = translate(input, t);
    assert.notEqual(translated, input);
    assert.doesNotMatch(translated, /forecast|conditions|readings|expected/);
    assert.deepEqual(translated.match(/\d+(?:[.:\-]\d+)*/g), input.match(/\d+(?:[.:\-]\d+)*/g));
  }
});
test("unknown API messages are preserved", () => {
  assert.equal(translate("Future provider message", t), "Future provider message");
});
