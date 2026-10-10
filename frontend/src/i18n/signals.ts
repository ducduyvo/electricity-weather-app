import type { TranslationKey } from ".";
import english from "./locales/en/weather.json";
const en = english.weather;

type Translator = (key: TranslationKey, values?: Record<string, string | number>) => string;
// Translate known English API templates at the presentation boundary.
// Unknown messages remain intact rather than inventing an interpretation.
export function translateSignalText(text: string, t: Translator): string {
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
    const values = Object.fromEntries(names.map((name, index) => [name, translateSignalText(match[index + 1], t)]));
    return t(`weather.${key}`, values) + translateSignalText(text.slice(match[0].length), t);
  }
  const suffix = " Some hourly readings are missing; this is the peak among available readings.";
  if (text.endsWith(suffix)) return translateSignalText(text.slice(0, -suffix.length), t) + translateSignalText(suffix, t);
  return text;
}
