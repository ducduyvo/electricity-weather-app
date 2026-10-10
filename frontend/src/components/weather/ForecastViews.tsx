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
  Legend,
} from "recharts";
import type { Forecast, ForecastView, Daily } from "../../types/weather";
import { localDateLabel } from "../../utils/forecastTime";
import { InfoTooltip } from "../common/InfoTooltip";
import { Panel } from "../common/Surface";

type HourlyChartPoint = Forecast["hourly"][number] & { label: string };
type DailyChartPoint = Daily & { label: string; solarEnergy: number | null };

export function ForecastViews({ view, data, chart, dailyChart, }: { view: ForecastView; data: Forecast; chart: HourlyChartPoint[]; dailyChart?: DailyChartPoint[]; }) {
  const { t, locale } = useI18n();
  return (
    <div className={`forecast-panels ${view}`}>
      {view === "hourly" ? (
        <>
          <Panel>
            <div>
              <Typography component="h2" variant="h2">{t("weather.temperature_next_72h")}</Typography>
              <p>{t("weather.2_m_ambient_air_temperature_c")}</p>
            </div>
            <ResponsiveContainer width="100%" height={280}>
              <LineChart data={chart}>
                <CartesianGrid
                  vertical={false}
                  stroke="var(--border)"
                  strokeDasharray="3 3"
                />
                <XAxis dataKey="label" minTickGap={45} />
                <YAxis />
                <Tooltip
                />
                <Line
                  dataKey="temperature" name={t("weather.temperature")}
                  stroke="var(--blue)"
                  type="monotone"
                  strokeWidth={3}
                  dot={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </Panel>
          <Panel>
            <div>
              <Typography component="h2" variant="h2">{t("weather.solar_irradiance_next_72h")}</Typography>
              <p className="chart-subtitle-with-info">{t("weather.global_horizontal_irradiance_w_m")}<InfoTooltip text={t("weather.solar_irradiance_is_the_amount_of_solar")} />
              </p>
            </div>
            <ResponsiveContainer width="100%" height={280}>
              <AreaChart data={chart}>
                <CartesianGrid
                  vertical={false}
                  stroke="var(--border)"
                  strokeDasharray="3 3"
                />
                <XAxis dataKey="label" minTickGap={45} />
                <YAxis />
                <Tooltip
                />
                <Area
                  dataKey="solarIrradiance" name={t("weather.solarirradiance")}
                  stroke="var(--blue)"
                  fill="var(--blue)"
                  type="monotone"
                  fillOpacity={0.15}
                  strokeWidth={2}
                />
              </AreaChart>
            </ResponsiveContainer>
          </Panel>
        </>
      ) : (
        <>
          <Panel>
            <div>
              <Typography component="h2" variant="h2">{t("weather.daily_forecast_details")}</Typography>
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
                    <TableCell>{t("weather.min_c")}</TableCell>
                    <TableCell>{t("weather.max_c")}</TableCell>
                    <TableCell>{t("weather.avg_c")}</TableCell>
                    <TableCell>
                      <span className="table-header-info">{t("weather.solar_kwh_m")}<InfoTooltip text={t("weather.total_solar_energy_received_per_square_metre")} />
                      </span>
                    </TableCell>

                    <TableCell>
                      <span className="table-header-info">{t("weather.peak_w_m_local_time")}<InfoTooltip text={t("weather.the_highest_forecast_solar_irradiance_during_the")} />
                      </span>
                    </TableCell>
                    <TableCell>{t("weather.coverage")}</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {data.daily.slice(0, 14).map((day) => (
                    <TableRow key={day.date}>
                      <TableCell component="th" scope="row">{localDateLabel(day.date, false, locale)}</TableCell>
                      <TableCell>{day.minTemperature?.toFixed(1) ?? "—"}</TableCell>
                      <TableCell>{day.maxTemperature?.toFixed(1) ?? "—"}</TableCell>
                      <TableCell>{day.avgTemperature?.toFixed(1) ?? "—"}</TableCell>
                      <TableCell>
                        {day.solarSampleCount > 0
                          ? day.solarEnergyKWhM2.toFixed(2)
                          : "—"}
                        {day.solarSampleCount > 0 &&
                          (!day.completeDay ||
                            day.solarSampleCount < day.hourCount) &&
                          t("weather.partial")}
                      </TableCell>
                      <TableCell>
                        {day.peakSolarIrradiance == null
                          ? "—"
                          : `${day.peakSolarIrradiance.toFixed(0)} / ${
                              day.peakSolarTime?.slice(11, 16) ?? "—"
                            }`}
                      </TableCell>
                      <TableCell>
                        {day.completeDay
                          ? t("weather.full_day")
                          : t("weather.partial_day", { hours: day.hourCount, expected: day.expectedHourCount })}
                        {day.temperatureSampleCount < day.hourCount &&
                          t("weather.missing_temperature")}
                        {day.solarSampleCount < day.hourCount &&
                          t("weather.missing_solar")}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Panel>
          <Panel>
            <div>
              <Typography component="h2" variant="h2">{t("weather.temperature_14_day_outlook")}</Typography>
              <p>{t("weather.daily_minimum_average_and_maximum_c")}</p>
            </div>
            <ResponsiveContainer width="100%" height={280}>
              <LineChart data={dailyChart}>
                <CartesianGrid
                  vertical={false}
                  stroke="var(--border)"
                  strokeDasharray="3 3"
                />
                <XAxis dataKey="label" minTickGap={15} />
                <YAxis />
                <Tooltip
                />
                <Legend />
                <Line
                  dataKey="minTemperature"
                  name={t("weather.minimum_c")}
                  stroke="var(--blue)"
                  strokeWidth={2}
                />
                <Line
                  dataKey="maxTemperature"
                  name={t("weather.maximum_c")}
                  stroke="var(--warning-text)"
                  strokeWidth={2}
                />
                <Line
                  dataKey="avgTemperature"
                  name={t("weather.average_c")}
                  stroke="var(--text-secondary)"
                  strokeDasharray="4 4"
                />
              </LineChart>
            </ResponsiveContainer>
          </Panel>
          <Panel>
            <div>
              <Typography component="h2" variant="h2">{t("weather.solar_availability_14_day_outlook")}</Typography>
              <p>{t("weather.daily_solar_energy_proxy_kwh_m")}</p>
            </div>
            <ResponsiveContainer width="100%" height={280}>
              <BarChart data={dailyChart}>
                <CartesianGrid
                  vertical={false}
                  stroke="var(--border)"
                  strokeDasharray="3 3"
                />
                <XAxis dataKey="label" minTickGap={15} />
                <YAxis />
                <Tooltip
                />
                <Bar
                  dataKey="solarEnergy"
                  name={t("weather.solar_proxy_kwh_m")}
                  fill="var(--blue)"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          </Panel>
        </>
      )}
    </div>
  );
}
