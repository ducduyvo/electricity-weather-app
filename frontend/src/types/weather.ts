export type WindPotential =
  | "very_low"
  | "low"
  | "moderate"
  | "high"
  | "strong_uncertain";
// Wind summaries and hourly wind fields use 120 m forecasts in m/s.
export type Daily = {
  windDirectionConsistency: number;
  meanWindDirection: number | null;
  windDirectionSampleCount: number;
  minWindSpeed: number | null;
  meanWindSpeed: number | null;
  maxWindSpeed: number | null;
  windSampleCount: number;
  date: string;
  minTemperature: number | null;
  maxTemperature: number | null;
  avgTemperature: number | null;
  solarEnergyKWhM2: number;
  peakSolarIrradiance: number | null;
  peakSolarTime?: string;
  solarSampleCount: number;
  hourCount: number;
  expectedHourCount: number;
  completeDay: boolean;
  temperatureSampleCount: number;
};
export type WeatherSignal = {
  kind: string;
  severity: string;
  title: string;
  detail: string;
};
export type Forecast = {
  timezone: string;
  hourly: {
    time: string;
    temperature: number | null;
    solarIrradiance: number | null;
    windSpeed: number | null;
    windDirection: number | null;
    windPotential: WindPotential | null;
  }[];
  daily: Daily[];
  signals: WeatherSignal[];
  outlookSignals: WeatherSignal[];
  fetchedAt: string;
};

export type ForecastView = "hourly" | "daily";
