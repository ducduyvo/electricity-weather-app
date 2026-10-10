import Alert from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import { translateSignalText } from "../../i18n/signals";
import { useI18n } from "../../i18n";
import type { ForecastView, WeatherSignal } from "../../types/weather";
import { Panel } from "../common/Surface";

export function TradingInsights({ view, itemsForView, }: { view: ForecastView; itemsForView?: WeatherSignal[]; }) {
  const { t } = useI18n();
  return (
    <Panel className="signals">
      <Typography component="h2" variant="h2">
        {view === "hourly"
          ? t("weather.short_term_trading_insights")
          : t("weather.14_day_trend_insights")}
      </Typography>
      <p className="insight-note">{t("weather.insight_disclaimer")}</p>
      {[
        {
          kinds: ["temperature"],
          title: t("weather.temperature_demand_conditions"),
        },
        { kinds: ["solar"], title: t("weather.solar_generation_conditions") },
        {
          kinds: ["combined", "weather"],
          title: t("weather.combined_weather_outlook"),
        },
      ].map((group) => {
        const items = (itemsForView ?? []).filter((signal) =>
          group.kinds.includes(signal.kind)
        );
        return (
          <div className="insight-group" key={group.title}>
            <Typography component="h3" variant="h3">{group.title}</Typography>
            {items.length > 0 ? (
              items.map((signal) => (
                <Alert icon={false} severity={signal.severity === "warning" ? "warning" : signal.severity === "normal" ? "success" : "info"} className={"signal " + signal.severity} key={translateSignalText(signal.title, t)}>
                  <b>{translateSignalText(signal.title, t)}</b>
                  <span>{translateSignalText(signal.detail, t)}</span>
                </Alert>
              ))
            ) : (
              <p>{t("weather.no_additional_insight_for_this_forecast_period")}</p>
            )}
          </div>
        );
      })}
    </Panel>
  );
}
