import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import english from "./locales/en/weather.json";
import finnish from "./locales/fi/weather.json";

export type TranslationKey = `weather.${keyof typeof english.weather}`;
const en = english.weather;
const fi: Record<keyof typeof en, string> = finnish.weather;
export type Locale = "en" | "fi";
type I18n = {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: TranslationKey, values?: Record<string, string | number>) => string;
};
const Context = createContext<I18n | null>(null);
export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocale] = useState<Locale>(() => {
    try { return localStorage.getItem("weather-locale") === "fi" ? "fi" : "en"; }
    catch { return "en"; }
  });
  useEffect(() => {
    document.documentElement.lang = locale;
    document.title = (locale === "fi" ? fi : en)["app_title"];
    try { localStorage.setItem("weather-locale", locale); } catch { /* Storage may be disabled. */ }
  }, [locale]);
  const t: I18n["t"] = (key, values = {}) => {
    const text = (locale === "fi" ? fi : en)[key.slice("weather.".length) as keyof typeof en] ?? key;
    return text.replace(/\{(\w+)\}/g, (placeholder, name: string) => String(values[name] ?? placeholder));
  };
  return <Context.Provider value={{ locale, setLocale, t }}>{children}</Context.Provider>;
}
export function useI18n() {
  const value = useContext(Context);
  if (!value) throw new Error("useI18n requires I18nProvider");
  return value;
}
