import type { Forecast } from "../types/weather";

const apiURL = (
  import.meta.env.VITE_API_URL ?? "http://localhost:8080"
).replace(/\/$/, "");

export async function fetchForecast(
  lat: number,
  lon: number,
  signal: AbortSignal
): Promise<Forecast> {
  const response = await fetch(`${apiURL}/api/weather?lat=${lat}&lon=${lon}`, {
    signal,
  });
  if (!response.ok) throw new Error("Weather request failed");
  return response.json() as Promise<Forecast>;
}
