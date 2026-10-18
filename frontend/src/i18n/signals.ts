import type { WeatherSignal } from "../types/weather";
import type { TranslationKey } from ".";
import english from "./locales/en/weather.json";
const en = english.weather;

type Translator = (
  key: TranslationKey,
  values?: Record<string, string | number>,
) => string;
// Translate known English API templates at the presentation boundary.
// Unknown messages remain intact rather than inventing an interpretation.
export function translateSignalText(
  text: string,
  t: Translator,
  locale?: string,
): string {
  // Forecast timestamps are local wall times; UTC here prevents browser timezone shifts.
  if (locale && /^\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2})?$/.test(text)) {
    const hasTime = text.length > 10;
    const date = new Date(hasTime ? `${text}:00Z` : `${text}T12:00:00Z`);
    if (!Number.isNaN(date.getTime())) {
      return new Intl.DateTimeFormat(locale === "en" ? "en-GB" : locale, {
        timeZone: "UTC",
        day: "numeric",
        month: "short",
        year: "numeric",
        ...(hasTime
          ? { hour: "2-digit", minute: "2-digit", hourCycle: "h23" as const }
          : {}),
      }).format(date);
    }
  }
  const exactKey = (Object.keys(en) as (keyof typeof en)[]).find(
    (key) => en[key] === text,
  );
  if (exactKey) return t(`weather.${exactKey}`);
  for (const key of Object.keys(en) as (keyof typeof en)[]) {
    const template = en[key];
    if (!template.includes("{")) continue;
    const names: string[] = [];
    const expression = template
      .split(/(\{\w+\})/)
      .map((part) => {
        if (part.startsWith("{") && part.endsWith("}")) {
          names.push(part.slice(1, -1));
          return "(.+?)";
        }
        return part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
      })
      .join("");
    const match = text.match(new RegExp("^" + expression));
    if (!match) continue;
    const values = Object.fromEntries(
      names.map((name, index) => [
        name,
        translateSignalText(match[index + 1], t, locale),
      ]),
    );
    return (
      t(`weather.${key}`, values) +
      translateSignalText(text.slice(match[0].length), t, locale)
    );
  }
  const suffix =
    " Some hourly readings are missing; this is the peak among available readings.";
  if (text.endsWith(suffix))
    return (
      translateSignalText(text.slice(0, -suffix.length), t, locale) +
      translateSignalText(suffix, t, locale)
    );
  return text;
}

// Structured codes identify interpretations without depending on English wording.
// Unknown codes and incomplete parameters retain the existing text fallback.
export function translateSignal(
  signal: WeatherSignal,
  t: Translator,
  locale: string,
): { title: string; detail: string } {
  const structured = translateTemperatureSolar(signal, t, locale);
  if (structured) return structured;
  const { change, start, end } = signal.params ?? {};
  const increasing = signal.code === "wind_speed_increasing";
  const decreasing = signal.code === "wind_speed_decreasing";
  const validDate = (value: unknown): value is string =>
    typeof value === "string" &&
    /^\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2})?$/.test(value) &&
    !Number.isNaN(
      new Date(
        value.length > 10 ? `${value}:00Z` : `${value}T12:00:00Z`,
      ).getTime(),
    );
  if (
    (increasing || decreasing) &&
    typeof change === "number" &&
    Number.isFinite(change) &&
    (increasing ? change > 0 : change < 0) &&
    validDate(start) &&
    validDate(end)
  ) {
    return {
      title: t(
        increasing
          ? "weather.wind_conditions_improving"
          : "weather.wind_conditions_weakening",
      ),
      detail: t("weather.wind_trend_detail", {
        change: new Intl.NumberFormat(locale, {
          minimumFractionDigits: 1,
          maximumFractionDigits: 1,
          signDisplay: "always",
        }).format(change),
        start: translateSignalText(start, t, locale),
        end: translateSignalText(end, t, locale),
        implication: t(
          increasing
            ? "weather.wind_increasing_implication"
            : "weather.wind_decreasing_implication",
        ),
      }),
    };
  }
  return {
    title: translateSignalText(signal.title, t, locale),
    detail: translateSignalText(signal.detail, t, locale),
  };
}

function translateTemperatureSolar(
  signal: WeatherSignal,
  t: Translator,
  locale: string,
): { title: string; detail: string } | null {
  const p = signal.params ?? {};
  const number = (value: unknown): value is number =>
    typeof value === "number" && Number.isFinite(value);
  const date = (value: unknown): value is string =>
    typeof value === "string" &&
    /^\d{4}-\d{2}-\d{2}$/.test(value) &&
    !Number.isNaN(new Date(`${value}T12:00:00Z`).getTime());
  const format = (value: number, digits: number, signed = false) =>
    new Intl.NumberFormat(locale, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
      signDisplay: signed ? "always" : "auto",
    }).format(value);
  const formatDate = (value: string) => translateSignalText(value, t, locale);
  const interpretations: Record<string, TranslationKey> = {
    heating_increase:
      "weather.colder_conditions_may_increase_heating_driven_electricity",
    cooling_ease:
      "weather.cooling_driven_electricity_demand_conditions_may_ease",
    cooling_increase:
      "weather.warmer_conditions_may_increase_cooling_driven_electricity",
    heating_ease:
      "weather.heating_driven_electricity_demand_conditions_may_ease",
    dependent: "weather.the_electricity_demand_implication_depends_on_heating",
    local_dependent:
      "weather.the_heating_cooling_demand_implication_depends_on",
  };
  const interpretation =
    typeof p.interpretation === "string" &&
    Object.prototype.hasOwnProperty.call(interpretations, p.interpretation)
      ? interpretations[p.interpretation]
      : undefined;
  if (
    (signal.code === "temperature_increasing" ||
      signal.code === "temperature_decreasing") &&
    number(p.change) &&
    interpretation
  ) {
    const rising = signal.code === "temperature_increasing";
    if (rising ? p.change <= 0 : p.change >= 0) return null;
    return {
      title: t(rising ? "weather.warmer_tomorrow" : "weather.colder_tomorrow"),
      detail:
        t(
          rising
            ? "weather.average_temperature_is_expected_to_rise_change"
            : "weather.average_temperature_is_expected_to_fall_change",
          { change: format(Math.abs(p.change), 1) },
        ) + t(interpretation),
    };
  }
  if (
    signal.code === "temperature_trend" &&
    number(p.change) &&
    date(p.start) &&
    date(p.end) &&
    interpretation
  ) {
    const titles: Record<string, TranslationKey> = {
      warming: "weather.warming_trend",
      cooling: "weather.cooling_trend",
      cold_spell: "weather.cold_spell_developing",
    };
    if (
      typeof p.trend !== "string" ||
      !Object.prototype.hasOwnProperty.call(titles, p.trend)
    )
      return null;
    if (p.trend === "warming" ? p.change <= 0 : p.change >= 0) return null;
    return {
      title: t(titles[p.trend]),
      detail:
        t("weather.daily_average_temperature_changes_by_change_c", {
          change: format(p.change, 1, true),
          start: formatDate(p.start),
          end: formatDate(p.end),
        }) + t(interpretation),
    };
  }
  if (
    (signal.code === "solar_daily_increasing" ||
      signal.code === "solar_daily_decreasing") &&
    number(p.change) &&
    date(p.date) &&
    date(p.baseline)
  ) {
    const rising = signal.code === "solar_daily_increasing";
    if (rising ? p.change <= 0 : p.change >= 0) return null;
    return {
      title: t(
        rising
          ? "weather.higher_solar_availability"
          : "weather.lower_solar_availability",
      ),
      detail:
        t(
          rising
            ? "weather.solar_energy_proxy_on_date_rises_change"
            : "weather.solar_energy_proxy_on_date_falls_change",
          {
            change: format(Math.abs(p.change), 2),
            date: formatDate(p.date),
            baseline: formatDate(p.baseline),
          },
        ) +
        t(
          rising
            ? "weather.higher_irradiance_may_improve_conditions_for_solar"
            : "weather.lower_irradiance_may_reduce_favorable_conditions_for",
        ),
    };
  }
  if (
    (signal.code === "solar_trend_increasing" ||
      signal.code === "solar_trend_decreasing") &&
    number(p.change) &&
    date(p.start) &&
    date(p.end)
  ) {
    const rising = signal.code === "solar_trend_increasing";
    if (rising ? p.change <= 0 : p.change >= 0) return null;
    return {
      title: t(
        rising
          ? "weather.solar_availability_improving"
          : "weather.solar_availability_weakening",
      ),
      detail:
        t("weather.the_daily_solar_energy_proxy_changes_by", {
          change: format(p.change, 2, true),
          start: formatDate(p.start),
          end: formatDate(p.end),
        }) +
        t(
          rising
            ? "weather.higher_irradiance_may_improve_conditions_for_solar"
            : "weather.lower_irradiance_may_weaken_conditions_for_solar",
        ),
    };
  }
  if (
    (signal.code === "solar_peak" || signal.code === "solar_peak_strong") &&
    number(p.peak) &&
    p.peak > 0 &&
    date(p.date) &&
    typeof p.time === "string" &&
    /^(?:[01]\d|2[0-3]):[0-5]\d$/.test(p.time) &&
    typeof p.partial === "boolean"
  ) {
    const strong = signal.code === "solar_peak_strong";
    return {
      title: t(
        strong
          ? "weather.strong_solar_period_expected"
          : "weather.solar_period_expected",
      ),
      detail:
        t("weather.solar_irradiance_on_date_is_forecast_to", {
          date: formatDate(p.date),
          time: p.time,
          peak: format(p.peak, 0),
          conditions: t(
            strong
              ? "weather.indicating_favorable_irradiance_conditions_for_solar_pv"
              : "weather.indicating_potential_solar_pv_generation_conditions",
          ),
        }) +
        (p.partial
          ? t("weather.some_hourly_readings_are_missing_this_is")
          : ""),
    };
  }
  return null;
}
