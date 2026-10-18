import {
  dailyChartValues,
  isDailyMetricComplete,
} from "../../utils/dailyForecast";
import { formatMeasurement } from "../../utils/measurement";
import { chartMetricColors } from "../../designTokens";
import Table from "@mui/material/Table";
import TableHead from "@mui/material/TableHead";
import TableBody from "@mui/material/TableBody";
import TableRow from "@mui/material/TableRow";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import Typography from "@mui/material/Typography";
import { useI18n } from "../../i18n";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  AreaChart,
  Area,
  BarChart,
  Bar,
} from "recharts";
import { ForecastTooltip } from "./ForecastTooltip";
import { DailyForecastLine, DailyForecastDot } from "./DailyForecastLine";
import { WindTooltip } from "./WindTooltip";
import type { Forecast, ForecastView, Daily } from "../../types/weather";
import { localDateLabel } from "../../utils/forecastTime";
import { InfoTooltip } from "../common/InfoTooltip";
import { Panel } from "../common/Surface";

type HourlyChartPoint = Forecast["hourly"][number] & { label: string };
type DailyChartPoint = Daily & { label: string; solarEnergy: number | null };

function DailyCoverageLegend({ metric }: { metric: "temperature" | "wind" }) {
  const { t } = useI18n();
  return (
    <p className="daily-coverage-legend">
      <span>
        <svg width="40" height="12" aria-hidden="true">
          <line
            x1="0"
            y1="6"
            x2="40"
            y2="6"
            stroke={chartMetricColors[metric]}
            strokeWidth="3"
          />
        </svg>
        {t("weather.chart_complete_day")}
      </span>
      <span>
        <svg width="40" height="12" aria-hidden="true">
          <line
            x1="0"
            y1="6"
            x2="40"
            y2="6"
            stroke={chartMetricColors[metric]}
            strokeWidth="3"
            strokeDasharray="5 5"
          />
        </svg>
        {t("weather.chart_partial_day")}
      </span>
    </p>
  );
}

export function ForecastViews({
  view,
  data,
  chart,
  dailyChart,
}: {
  view: ForecastView;
  data: Forecast;
  chart: HourlyChartPoint[];
  dailyChart?: DailyChartPoint[];
}) {
  const { t, locale } = useI18n();
  const hourlyTicks = chart
    .filter((point) => /T(00|12):00$/.test(point.time))
    .map((point) => point.label);
  const maximumSolar = Math.max(
    0,
    ...chart.map((point) =>
      Number.isFinite(point.solarIrradiance) ? point.solarIrradiance ?? 0 : 0,
    ),
  );
  const solarStep =
    maximumSolar <= 250
      ? 50
      : maximumSolar <= 500
      ? 100
      : maximumSolar <= 1000
      ? 200
      : 500;
  const solarAxisMaximum = Math.max(
    solarStep,
    Math.ceil(maximumSolar / solarStep) * solarStep,
  );
  const solarTicks = Array.from(
    { length: solarAxisMaximum / solarStep + 1 },
    (_, index) => index * solarStep,
  );

  const maximumWind = Math.max(
    12,
    ...(dailyChart ?? []).map((day) =>
      Number.isFinite(day.maxWindSpeed) ? day.maxWindSpeed ?? 0 : 0,
    ),
  );
  const windAxisMaximum = Math.ceil(maximumWind / 3) * 3;
  const windTicks = Array.from(
    { length: windAxisMaximum / 3 + 1 },
    (_, index) => index * 3,
  );

  return (
    <section aria-labelledby="forecast-details-heading">
      <Typography
        id="forecast-details-heading"
        component="h2"
        variant="h2"
        className="forecast-section-heading"
      >
        {t(view === "hourly" ? "weather.hourly_details" : "weather.daily_view")}
      </Typography>
      <div className={`forecast-panels ${view}`}>
        {view === "hourly" ? (
          <>
            <Panel>
              <div>
                <Typography component="h2" variant="h2">
                  {t("weather.temperature_next_72h")}
                </Typography>
                <p>{t("weather.2_m_ambient_air_temperature_c")}</p>
              </div>
              {chart.every((p) => p.temperature == null) ? (
                <p>{t("weather.chart_no_measurements")}</p>
              ) : (
                <ResponsiveContainer width="100%" height={280}>
                  <LineChart data={chart}>
                    <CartesianGrid
                      vertical={false}
                      stroke="var(--border)"
                      strokeDasharray="3 3"
                    />
                    <XAxis
                      dataKey="label"
                      ticks={hourlyTicks}
                      interval={0}
                      padding={{ left: 40, right: 40 }}
                      tickMargin={12}
                      height={40}
                    />
                    <YAxis />
                    <Tooltip
                      trigger="hover"
                      content={<ForecastTooltip metric="temperature" />}
                      filterNull={false}
                    />
                    <Line
                      activeDot={{ r: 4 }}
                      connectNulls={false}
                      dataKey="temperature"
                      name={t("weather.temperature")}
                      stroke={chartMetricColors.temperature}
                      type="monotone"
                      strokeWidth={2}
                      dot={false}
                    />
                  </LineChart>
                </ResponsiveContainer>
              )}
            </Panel>
            <Panel>
              <div>
                <Typography component="h2" variant="h2">
                  {t("weather.wind_hourly_title")}
                </Typography>
                <p className="chart-subtitle-with-info">
                  {t("weather.wind_hourly_description")}
                  <InfoTooltip text={t("weather.wind_explanation")} />
                </p>
              </div>
              {chart.every((p) => p.windSpeed == null) ? (
                <p>{t("weather.chart_no_measurements")}</p>
              ) : (
                <ResponsiveContainer width="100%" height={280}>
                  <LineChart data={chart}>
                    <CartesianGrid
                      vertical={false}
                      stroke="var(--border)"
                      strokeDasharray="3 3"
                    />
                    <XAxis
                      dataKey="label"
                      ticks={hourlyTicks}
                      interval={0}
                      padding={{ left: 40, right: 40 }}
                      tickMargin={12}
                      height={40}
                    />
                    <YAxis domain={[0, "auto"]} />
                    <Tooltip
                      trigger="hover"
                      content={<WindTooltip />}
                      filterNull={false}
                    />
                    <Line
                      activeDot={{ r: 4 }}
                      dataKey="windSpeed"
                      name={t("weather.wind_speed")}
                      stroke={chartMetricColors.wind}
                      strokeWidth={2}
                      dot={false}
                      connectNulls={false}
                    />
                  </LineChart>
                </ResponsiveContainer>
              )}
            </Panel>
            <Panel className="hourly-solar-panel">
              <div>
                <Typography component="h2" variant="h2">
                  {t("weather.solar_irradiance_next_72h")}
                </Typography>
                <p className="chart-subtitle-with-info">
                  {t("weather.global_horizontal_irradiance_w_m")}
                  <InfoTooltip
                    text={t("weather.solar_irradiance_is_the_amount_of_solar")}
                  />
                </p>
              </div>
              {chart.every((p) => p.solarIrradiance == null) ? (
                <p>{t("weather.chart_no_measurements")}</p>
              ) : (
                <ResponsiveContainer width="100%" height={280}>
                  <AreaChart data={chart}>
                    <CartesianGrid
                      vertical={false}
                      stroke="var(--border)"
                      strokeDasharray="3 3"
                    />
                    <XAxis
                      dataKey="label"
                      ticks={hourlyTicks}
                      interval={0}
                      padding={{ left: 40, right: 40 }}
                      tickMargin={12}
                      height={40}
                    />
                    <YAxis domain={[0, solarAxisMaximum]} ticks={solarTicks} />
                    <Tooltip
                      content={<ForecastTooltip metric="solar" />}
                      filterNull={false}
                    />
                    <Area
                      connectNulls={false}
                      dataKey="solarIrradiance"
                      name={t("weather.solarirradiance")}
                      stroke={chartMetricColors.solar}
                      fill={chartMetricColors.solar}
                      type="monotone"
                      fillOpacity={0.15}
                      strokeWidth={2}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </Panel>
          </>
        ) : (
          <>
            <Panel>
              <div>
                <Typography component="h2" variant="h2">
                  {t("weather.daily_forecast_details")}
                </Typography>
                <p>
                  {data.daily.length < 14
                    ? t("weather.available_days", { count: data.daily.length })
                    : t("weather.up_to_14_days_no_missing_values")}
                </p>
              </div>
              <TableContainer
                className="table-scroll"
                role="region"
                aria-label={t("weather.daily_forecast_table")}
                tabIndex={0}
              >
                <Table size="small" stickyHeader>
                  <TableHead>
                    <TableRow>
                      <TableCell>{t("weather.date")}</TableCell>
                      <TableCell>{t("weather.table_temperature")}</TableCell>
                      <TableCell>
                        <span className="table-header-info">
                          {t("weather.solar_kwh_m")}
                          <InfoTooltip
                            text={t(
                              "weather.total_solar_energy_received_per_square_metre",
                            )}
                          />
                        </span>
                      </TableCell>

                      <TableCell>
                        <span className="table-header-info">
                          {t("weather.peak_w_m_local_time")}
                          <InfoTooltip
                            text={t(
                              "weather.the_highest_forecast_solar_irradiance_during_the",
                            )}
                          />
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="table-header-info">
                          {t("weather.table_wind")}
                          <InfoTooltip text={t("weather.wind_explanation")} />
                        </span>
                      </TableCell>
                      <TableCell>{t("weather.coverage")}</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {data.daily
                      .slice(0, 14)
                      .map(dailyChartValues)
                      .map((day) => (
                        <TableRow key={day.date}>
                          <TableCell component="th" scope="row">
                            {localDateLabel(day.date, false, locale)}
                          </TableCell>
                          <TableCell>
                            <span className="daily-metric-primary">
                              {formatMeasurement(day.avgTemperature, "")}{" "}
                              {t("weather.metric_avg")}
                              {day.avgTemperature != null &&
                                !isDailyMetricComplete(day, "temperature") &&
                                t("weather.partial")}
                            </span>
                            <span className="daily-metric-secondary">
                              {formatMeasurement(day.minTemperature, "")}{" "}
                              {t("weather.metric_min")} ·{" "}
                              {formatMeasurement(day.maxTemperature, "")}{" "}
                              {t("weather.metric_max")}
                            </span>
                          </TableCell>
                          <TableCell>
                            {day.solarSampleCount > 0 &&
                            Number.isFinite(day.solarEnergyKWhM2)
                              ? day.solarEnergyKWhM2.toFixed(2)
                              : "—"}
                            {day.solarSampleCount > 0 &&
                              !isDailyMetricComplete(day, "solar") &&
                              t("weather.partial")}
                          </TableCell>
                          <TableCell>
                            {day.solarSampleCount === 0 ||
                            day.peakSolarIrradiance == null ||
                            !Number.isFinite(day.peakSolarIrradiance)
                              ? "—"
                              : `${day.peakSolarIrradiance.toFixed(0)} / ${
                                  day.peakSolarTime?.slice(11, 16) ?? "—"
                                }`}
                          </TableCell>
                          <TableCell>
                            <span className="daily-metric-primary">
                              {formatMeasurement(day.avgWindSpeed, "")}{" "}
                              {t("weather.metric_avg")}
                              {day.windSampleCount > 0 &&
                                !isDailyMetricComplete(day, "wind") &&
                                t("weather.partial")}
                            </span>
                            <span className="daily-metric-secondary">
                              {formatMeasurement(day.minWindSpeed, "")}{" "}
                              {t("weather.metric_min")} ·{" "}
                              {formatMeasurement(day.maxWindSpeed, "")}{" "}
                              {t("weather.metric_max")}
                            </span>
                            {day.avgWindDirection != null &&
                              Number.isFinite(day.avgWindDirection) && (
                                <span
                                  className="daily-wind-direction daily-metric-secondary"
                                  title={t("weather.daily_wind_from", {
                                    direction: [
                                      "N",
                                      "NE",
                                      "E",
                                      "SE",
                                      "S",
                                      "SW",
                                      "W",
                                      "NW",
                                    ][
                                      Math.round(day.avgWindDirection / 45) % 8
                                    ],
                                  })}
                                >
                                  <svg
                                    width="16"
                                    height="16"
                                    viewBox="0 0 16 16"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="1.5"
                                    aria-hidden="true"
                                    style={{
                                      transform: `rotate(${
                                        (day.avgWindDirection + 180) % 360
                                      }deg)`,
                                    }}
                                  >
                                    <path d="M8 14V2M3 7l5-5 5 5" />
                                  </svg>
                                  {t("weather.table_prevailing_wind", {
                                    direction: [
                                      "N",
                                      "NE",
                                      "E",
                                      "SE",
                                      "S",
                                      "SW",
                                      "W",
                                      "NW",
                                    ][
                                      Math.round(day.avgWindDirection / 45) % 8
                                    ],
                                    degrees:
                                      Math.round(day.avgWindDirection) % 360,
                                  })}
                                </span>
                              )}
                          </TableCell>
                          <TableCell>
                            {day.completeDay
                              ? t("weather.full_day")
                              : t("weather.partial_day", {
                                  hours: day.hourCount,
                                  expected: day.expectedHourCount,
                                })}
                            {day.temperatureSampleCount < day.hourCount &&
                              t("weather.missing_temperature")}
                            {day.solarSampleCount < day.hourCount &&
                              t("weather.missing_solar")}
                            {day.windSampleCount < day.hourCount &&
                              t("weather.wind_missing_coverage")}
                          </TableCell>
                        </TableRow>
                      ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Panel>
            <Panel>
              <div>
                <Typography component="h2" variant="h2">
                  {t("weather.temperature_14_day_outlook")}
                </Typography>
                <p>{t("weather.daily_minimum_average_and_maximum_c")}</p>
              </div>
              <div className="outlook-chart-scroll">
                <div className="outlook-line-chart">
                  {(dailyChart ?? []).every((d) => d.avgTemperature == null) ? (
                    <p>{t("weather.chart_no_measurements")}</p>
                  ) : (
                    <ResponsiveContainer width="100%" height={280}>
                      <LineChart
                        data={dailyChart}
                        margin={{ top: 8, right: 24, bottom: 8, left: 8 }}
                      >
                        <CartesianGrid
                          vertical={false}
                          stroke="var(--border)"
                          strokeDasharray="3 3"
                        />
                        <XAxis
                          dataKey="label"
                          interval={0}
                          tickMargin={12}
                          padding={{ left: 24, right: 24 }}
                        />
                        <YAxis width={72} tickMargin={12} />
                        <Tooltip
                          trigger="hover"
                          content={
                            <ForecastTooltip metric="temperature" daily />
                          }
                          filterNull={false}
                        />
                        <Line
                          activeDot={
                            <DailyForecastDot metric="temperature" r={4} />
                          }
                          connectNulls={false}
                          dataKey="avgTemperature"
                          shape={
                            <DailyForecastLine
                              days={dailyChart ?? []}
                              metric="temperature"
                            />
                          }
                          dot={<DailyForecastDot metric="temperature" r={3} />}
                          name={t("weather.chart_average")}
                          stroke={chartMetricColors.temperature}
                          strokeWidth={3}
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  )}
                </div>
              </div>
              <DailyCoverageLegend metric="temperature" />
            </Panel>
            <Panel>
              <div>
                <Typography component="h2" variant="h2">
                  {t("weather.wind_daily_title")}
                </Typography>
                <p className="chart-subtitle-with-info">
                  {t("weather.wind_daily_description")}
                  <InfoTooltip text={t("weather.wind_explanation")} />
                </p>
              </div>
              <div className="outlook-chart-scroll">
                <div className="outlook-line-chart">
                  {(dailyChart ?? []).every((d) => d.avgWindSpeed == null) ? (
                    <p>{t("weather.chart_no_measurements")}</p>
                  ) : (
                    <ResponsiveContainer width="100%" height={280}>
                      <LineChart
                        data={dailyChart}
                        margin={{ top: 8, right: 24, bottom: 8, left: 8 }}
                      >
                        <CartesianGrid
                          vertical={false}
                          stroke="var(--border)"
                          strokeDasharray="3 3"
                        />
                        <XAxis
                          dataKey="label"
                          interval={0}
                          tickMargin={12}
                          padding={{ left: 24, right: 24 }}
                        />
                        <YAxis
                          domain={[0, windAxisMaximum]}
                          ticks={windTicks}
                          width={88}
                          tickMargin={12}
                        />
                        <Tooltip
                          trigger="hover"
                          content={<ForecastTooltip metric="wind" daily />}
                          filterNull={false}
                        />
                        <Line
                          activeDot={<DailyForecastDot metric="wind" r={4} />}
                          dataKey="avgWindSpeed"
                          shape={
                            <DailyForecastLine
                              days={dailyChart ?? []}
                              metric="wind"
                            />
                          }
                          dot={<DailyForecastDot metric="wind" r={3} />}
                          name={t("weather.chart_average")}
                          stroke={chartMetricColors.wind}
                          strokeWidth={3}
                          connectNulls={false}
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  )}
                </div>
              </div>
              <DailyCoverageLegend metric="wind" />
            </Panel>
            <Panel>
              <div>
                <Typography component="h2" variant="h2">
                  {t("weather.solar_availability_14_day_outlook")}
                </Typography>
                <p>{t("weather.daily_solar_energy_proxy_kwh_m")}</p>
              </div>
              <div className="outlook-chart-scroll">
                <div className="outlook-line-chart">
                  {(dailyChart ?? []).every((d) => d.solarEnergy == null) ? (
                    <p>{t("weather.chart_no_measurements")}</p>
                  ) : (
                    <ResponsiveContainer width="100%" height={280}>
                      <BarChart
                        data={dailyChart}
                        margin={{ top: 8, right: 24, bottom: 8, left: 8 }}
                      >
                        <CartesianGrid
                          vertical={false}
                          stroke="var(--border)"
                          strokeDasharray="3 3"
                        />
                        <XAxis dataKey="label" interval={0} tickMargin={12} />
                        <YAxis width={72} tickMargin={12} />
                        <Tooltip
                          content={<ForecastTooltip metric="solar" daily />}
                          filterNull={false}
                        />
                        <Bar
                          dataKey="solarEnergy"
                          name={t("weather.solar_proxy_kwh_m")}
                          fill={chartMetricColors.solar}
                          radius={[4, 4, 0, 0]}
                        />
                      </BarChart>
                    </ResponsiveContainer>
                  )}
                </div>
              </div>
            </Panel>
          </>
        )}
      </div>
    </section>
  );
}
