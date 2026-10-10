import { useI18n } from "../../i18n";
import type { Forecast } from "../../types/weather";
import { localDateLabel } from "../../utils/forecastTime";
import { StatCard } from "../common/Surface";

export function DailyCards({ data }: { data: Forecast }) {
  const { t, locale } = useI18n();
  return (
    <section className="cards">
      {data.daily.slice(0, 3).map((d) => (
        <StatCard key={d.date}>
          <span>{localDateLabel(d.date, false, locale).toUpperCase()}</span>
          <strong>{d.avgTemperature?.toFixed(1) ?? "—"}°C</strong>
          <small>
            {d.minTemperature?.toFixed(0) ?? "—"}° /{" "}
            {d.maxTemperature?.toFixed(0) ?? "—"}° ·{" "}
            {d.solarSampleCount > 0 ? d.solarEnergyKWhM2.toFixed(2) : "—"}{" "}
            {t("weather.kwh_m_solar_proxy")}
            {d.solarSampleCount > 0 &&
              (!d.completeDay || d.solarSampleCount < d.hourCount) &&
              t("weather.partial")}
          </small>
        </StatCard>
      ))}
    </section>
  );
}
