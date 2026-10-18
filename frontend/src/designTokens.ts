// Shared values for the MUI theme and CSS custom properties.
export const colors = {
  chartTemperature: "#BF1506",
  chartSolar: "#CA8A04",
  chartWind: "#2563EB",
  navy: "#222e46",
  navyHover: "#4c5d80",
  blue: "#046ad1",
  infoTitle: "#004683",
  blueLight: "#f2f9ff",
  surface: "#ffffff",
  background: "#f9fafa",
  border: "#e1e4e8",
  borderStrong: "#878e94",
  text: "#373f46",
  textSecondary: "#505a63",
  textMuted: "#697178",
  yellow: "#ffc126",
  warningBackground: "#fff8e5",
  warningText: "#8e6e01",
  errorBackground: "#feeced",
  errorText: "#ba2225",
  accent: "#a3b1cc",
  tableHeader: "#eceff1",
} as const;

export const fontFamily =
  'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif';

export const borderRadius = 4;

export const cssTokens = {
  "--radius": `${borderRadius}px`,
  "--info-title": colors.infoTitle,
  "--chart-temperature": colors.chartTemperature,
  "--chart-solar": colors.chartSolar,
  "--chart-wind": colors.chartWind,
  "--font-family": fontFamily,
  "--navy": colors.navy,
  "--navy-hover": colors.navyHover,
  "--blue": colors.blue,
  "--blue-light": colors.blueLight,
  "--surface": colors.surface,
  "--background": colors.background,
  "--border": colors.border,
  "--border-strong": colors.borderStrong,
  "--text": colors.text,
  "--text-secondary": colors.textSecondary,
  "--text-muted": colors.textMuted,
  "--yellow": colors.yellow,
  "--warning-bg": colors.warningBackground,
  "--warning-text": colors.warningText,
  "--error-bg": colors.errorBackground,
  "--error-text": colors.errorText,
  "--accent": colors.accent,
  "--table-header-bg": colors.tableHeader,
};

export const chartMetricColors = {
  temperature: "var(--chart-temperature)",
  solar: "var(--chart-solar)",
  wind: "var(--chart-wind)",
} as const;
