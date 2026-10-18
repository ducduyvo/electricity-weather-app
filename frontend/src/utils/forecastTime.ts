// Provider timestamps are local wall times. Format them without shifting them
// into the browser timezone; the API's IANA timezone is displayed beside charts.
export function localDateLabel(
  value: string,
  hourly = false,
  locale = "en",
): string {
  return new Date(
    value.length === 10 ? `${value}T12:00:00Z` : `${value}:00Z`,
  ).toLocaleString(locale, {
    timeZone: "UTC",
    weekday: "short",
    ...(hourly
      ? { hour: "2-digit", minute: "2-digit", hourCycle: "h23" }
      : { month: "short", day: "numeric" }),
  });
}

export function currentLocalHour(timezone: string): string {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    hourCycle: "h23",
  }).formatToParts(new Date());
  const part = (name: string) =>
    parts.find((value) => value.type === name)?.value;
  return `${part("year")}-${part("month")}-${part("day")}T${part("hour")}:00`;
}

// Subtract the local minutes/seconds from an instant, preserving repeated hours
// and fractional-hour timezone offsets.
export function currentForecastHour(
  timezone: string,
  now = new Date(),
): number {
  const parts = new Intl.DateTimeFormat("en", {
    timeZone: timezone,
    minute: "2-digit",
    second: "2-digit",
  }).formatToParts(now);
  const minute = Number(parts.find((p) => p.type === "minute")?.value);
  const second = Number(parts.find((p) => p.type === "second")?.value);
  return Math.floor(now.getTime() / 1000) - minute * 60 - second;
}

export function hourlyPointLabel(
  unixTime: number,
  timezone: string,
  locale: string,
  repeated = false,
): string {
  return new Intl.DateTimeFormat(locale, {
    timeZone: timezone,
    weekday: "short",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    ...(repeated ? { timeZoneName: "shortOffset" as const } : {}),
  }).format(new Date(unixTime * 1000));
}
