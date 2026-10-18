// Metric symbols are decorative; adjacent text labels identify each value.
export function WeatherMetricIcon({ kind }: { kind: "solar" | "wind" | "partly-cloudy" }) {
  return (
    <svg
      className="weather-metric-icon"
      viewBox="0 0 24 24"
      width="20"
      height="20"
      aria-hidden="true"
      focusable="false"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {kind === "partly-cloudy" ? (
        <>
          <path d="M13 3V1M19 5l1.5-1.5M21 10h2M7 5 5.5 3.5" />
          <path d="M10 7a4 4 0 0 1 7 4" />
          <path d="M7 20h9a4 4 0 0 0 0-8h-1a5 5 0 0 0-9-1 4.5 4.5 0 0 0 1 9Z" />
        </>
      ) : kind === "solar" ? (
        <>
          <circle cx="12" cy="12" r="4" />
          <path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5" />
        </>
      ) : (
        <>
          <path d="M3 8h11a3 3 0 1 0-3-3M3 12h16a3 3 0 1 1-3 3M3 16h6a3 3 0 1 1-3 3" />
        </>
      )}
    </svg>
  );
}
