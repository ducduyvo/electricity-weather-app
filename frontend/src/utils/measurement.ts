// Zero is a valid value; unavailable/non-finite measurements have no unit.
export function formatMeasurement(
  value: number | null | undefined,
  unit: string,
  digits = 1,
): string {
  return value != null && Number.isFinite(value)
    ? `${value.toFixed(digits)}${unit}`
    : "—";
}
