export type Daily = {
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
  }[];
  daily: Daily[];
  signals: WeatherSignal[];
  outlookSignals: WeatherSignal[];
  fetchedAt: string;
};

export type ForecastView = "hourly" | "daily";
