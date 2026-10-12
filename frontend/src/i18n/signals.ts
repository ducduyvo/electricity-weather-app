import type { TranslationKey } from ".";
import english from "./locales/en/weather.json";
const en = english.weather;

type Translator = (key: TranslationKey, values?: Record<string, string | number>) => string;
// Translate known English API templates at the presentation boundary.
// Unknown messages remain intact rather than inventing an interpretation.
export function translateSignalText(text: string, t: Translator, locale?: string): string {
  // Forecast timestamps are local wall times; UTC here prevents browser timezone shifts.
  if (locale && /^\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2})?$/.test(text)) {
    const hasTime = text.length > 10;
    const date = new Date(hasTime ? `${text}:00Z` : `${text}T12:00:00Z`);
    if (!Number.isNaN(date.getTime())) {
      return new Intl.DateTimeFormat(locale === "en" ? "en-GB" : locale, {
        timeZone: "UTC", day: "numeric", month: "short", year: "numeric",
        ...(hasTime ? { hour: "2-digit", minute: "2-digit", hourCycle: "h23" as const } : {}),
      }).format(date);
    }
  }
  const exactKey = (Object.keys(en) as (keyof typeof en)[]).find((key) => en[key] === text);
  if (exactKey) return t(`weather.${exactKey}`);
  for (const key of Object.keys(en) as (keyof typeof en)[]) {
    const template = en[key];
    if (!template.includes("{")) continue;
    const names: string[] = [];
    const expression = template.split(/(\{\w+\})/).map((part) => {
      if (part.startsWith("{") && part.endsWith("}")) {
        names.push(part.slice(1, -1));
        return "(.+?)";
      }
      return part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    }).join("");
    const match = text.match(new RegExp("^" + expression));
    if (!match) continue;
    const values = Object.fromEntries(names.map((name, index) => [name, translateSignalText(match[index + 1], t, locale)]));
    return t(`weather.${key}`, values) + translateSignalText(text.slice(match[0].length), t, locale);
  }
  const suffix = " Some hourly readings are missing; this is the peak among available readings.";
  if (text.endsWith(suffix)) return translateSignalText(text.slice(0, -suffix.length), t, locale) + translateSignalText(suffix, t, locale);
  return text;
}
