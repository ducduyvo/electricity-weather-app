# Weather Trading Monitor

A small full-stack dashboard for an electricity retailer. It deliberately prioritizes **temperature** and **solar irradiance**, then converts large day-on-day weather changes into cautious trading-context signals.

## Run

```bash
docker compose up --build
```

Open `http://localhost:5173`. Backend health: `http://localhost:8080/api/health`.

Without Docker: `cd backend && go run ./cmd/server`, then `cd frontend && npm ci && npm run dev`.

## Architecture

`React/TypeScript -> Weather Handler -> Provider -> Cached -> OpenMeteo -> Open-Meteo API`

The browser does not depend directly on the provider response. The Go layer validates inputs, owns the provider adapter, maps data into a small application model, derives daily summaries/signals, applies a 10-minute in-memory cache, and converts upstream failure to a controlled 503 response. `Provider` makes a future FMI/ECMWF adapter possible without changing handlers/UI.

## Assumptions

- One coordinate is an acceptable first approximation for an area. A production retailer may need bidding-zone or portfolio-weighted weather.
- The hourly view shows the next 72 available hourly readings from the current hour in the forecast timezone. The 14-day outlook shows up to seven daily aggregates from the same forecast.
- `temperature_2m` represents ambient temperature relevant to heating/cooling demand.
- `shortwave_radiation` is used as Global Horizontal Irradiance (GHI), a useful high-level proxy for solar availability. It is **not** a forecast of a specific PV fleet.
- Provider-local time is requested (`timezone=auto`). The backend does not manually apply UTC offsets.
- Weather signals are decision support, not MW forecasts. Portfolio composition, heating type, PV capacity, load history and market conditions are intentionally not inferred.

## Decision logic

For each local calendar day the backend calculates average/min/max temperature and integrates hourly GHI into approximate daily kWh/m² (`W/m² / 1000` for each hourly interval).

Signals compare tomorrow with today:

- average temperature drop >= 3°C -> `Colder tomorrow` / heating-driven demand may increase
- average temperature rise >= 3°C -> `Warmer tomorrow` / heating-driven demand may ease
- daily solar proxy drop >= 0.5 kWh/m² -> `Lower solar availability`
- daily solar proxy rise >= 0.5 kWh/m² -> `Higher solar availability`
- otherwise, when both comparisons have sufficient data -> stable weather outlook
- a tomorrow solar peak >= 500 W/m² -> strong solar-period insight with local peak time
- a significant temperature fall below 18°C plus a solar decline -> potentially tighter weather-driven conditions
- tomorrow averaging 18–22°C plus higher solar availability -> potentially more favorable weather-driven conditions

The 18°C heating reference, 22°C cooling reference, and 500 W/m² strong-solar
label are simple descriptive heuristics, not locally calibrated demand or PV
models. Day-on-day temperature changes drive the temperature signals; the
reference bands only qualify their heating/cooling interpretation. In hot
conditions a temperature rise can increase cooling-demand conditions rather
than always implying lower demand.

Solar peak value/time and sample counts are returned with each daily summary.
The peak is tomorrow's highest available hourly reading (the first occurrence
wins ties); a single-day forecast describes that available date instead.
Peak times use the provider-local timestamp without browser timezone conversion.
Energy sums remain hourly irradiance proxies in kWh/m², not electricity MWh.
Solar comparisons are suppressed when either day's hourly solar data is
incomplete; zero solar energy is a valid baseline. Partial data can still provide a
peak insight, clearly marked as the peak among available readings.
Missing/insufficient comparison data produces a data-quality insight rather
than a misleading stable-weather claim.

The language intentionally says *may* rather than claiming a causal MW impact that the available data cannot support.

## Edge cases

- Invalid latitude/longitude -> HTTP 400.
- Provider timeout/non-200 -> HTTP 503 and a retry state in the UI.
- HTTP client has a 5-second timeout.
- Null weather values remain null; they are not converted to zero. Missing temperature values are excluded from aggregates.
- Zero irradiance is valid at night and is preserved.
- Solar changes use signed differences in kWh/m², so zero-energy baselines remain valid.
- The application uses the provider's IANA timezone rather than hard-coded offsets, avoiding normal DST offset mistakes.
- Repeated requests for the same rounded coordinate are cached for 10 minutes to reduce upstream calls.

## One improvement with more time

I would turn this from a weather monitor into a **portfolio-aware load and generation forecast**. I would combine weather with historical customer load, calendar effects, heating technology/temperature sensitivity, and installed PV capacity. The UI could then show an expected MW demand/solar range and uncertainty rather than only qualitative weather signals. Forecast error/backtesting would be a first-class metric so traders can see how much confidence to place in the model.

## Testing

```bash
cd backend
go test ./...
go test -race ./...
```

The backend tests cover handler status codes and JSON responses, coordinate validation (including non-finite values), request context propagation, upstream mapping/errors/null values/cancellation with local HTTP test servers, cache hits/expiry/failures/concurrency, configuration, daily aggregation, and signal thresholds. Tests do not call the live weather API. Frontend automated UI tests remain a future improvement.

Build the frontend with `cd frontend && npm ci && npm run build`. Dependency versions are pinned and the lockfile is committed.

The frontend defaults to `http://localhost:8080`. Set `VITE_API_URL` to another backend URL, or to an empty string for relative `/api/weather` requests behind a same-origin proxy. Vite reads this setting at startup/build time. Superseded requests are aborted and cannot update data, loading, or errors.

See [backend/README.md](backend/README.md) for backend configuration and lifecycle details.

## Engineering choices / trade-offs

This is intentionally small: no database, authentication, Kubernetes, message queue or persistent cache. None is needed for a read-only take-home with public weather data. If usage became multi-instance/high-volume, Redis could replace the in-memory cache. Observability (structured logs, request metrics, provider latency/error rate) would be the next operational addition.


## Manually checking trading insights

Run `docker compose up --build` and open `http://localhost:5173`.
Select a city and inspect the Temperature / demand conditions, Solar generation
conditions, and Combined weather outlook groups below the daily weather cards.
The solar insight includes the forecast date, local peak time, and W/m² value.
The charts remain the primary hourly weather displays. Change cities rapidly:
old requests must not overwrite the selected city's data. Stop the backend to
check the error/retry state, then restart it and retry.

Inspect raw summaries and signals with:

```sh
curl 'http://localhost:8080/api/weather?lat=60.1699&lon=24.9384'
```

Live forecasts may not trigger every threshold. Deterministic Go tests cover
cold/warm changes, cooling/heating wording, higher/lower solar availability,
peak detection, combined outlooks, missing values, partial solar readings,
zero baselines, and insufficient forecast days.


## Forecast resolutions

Use **Next 3-day** for hourly temperature/irradiance and short-term insights.
Use **14-day outlook** for daily minimum/average/maximum temperature, daily solar
energy, a coverage table with local peak times, and broader trend insights.
The daily charts contain at most seven points/bars, not 168 hourly points.
Switching views does not trigger another API request.

The existing Open-Meteo request already uses `forecast_days=7` and `timezone=auto`.
Go groups valid hourly timestamps by the provider-local calendar date, sorts
summaries by date, and exposes day/sample coverage. Expected day length is
calculated in the provider's IANA timezone, including 23/25-hour DST days.
Timezone data is embedded in the Go binary so this also works in the small
Docker runtime. Invalid timestamps are excluded from daily aggregation.
If the provider gives fewer days, only those days are shown.

Missing temperatures stay null and are excluded from averages/min/max. Missing
solar readings stay missing; partial sums are labeled in the table and omitted
from the daily solar chart. Incomplete first/last days remain visible with
coverage counts, but do not contribute to weekly trend signals. No missing
hours or days are interpolated. Peaks on incomplete days are peaks among the
available readings.

Weekly insights are returned in the additive `outlookSignals` field; existing
`signals` continue to describe short-term conditions. Trends require at least
three consecutive complete days, at least two changes in one direction, and no
reversal. Temperature change must reach 3°C; solar change must reach 0.5 kWh/m² in magnitude. Missing dates/readings interrupt the relevant trend.
Cold/warm interpretation uses the existing descriptive heating/cooling bands.
These are weather-derived conditions, not demand, generation, or price forecasts.

To check both views, run `docker compose up --build`, open
`http://localhost:5173`, and switch between the two forecast buttons. The hourly
charts retain detailed values, while the daily view has min/average/max lines,
solar bars, and date/coverage details. Both display the forecast timezone.
Threshold-dependent insights vary with real weather; Go tests provide fixed
seven-day, missing-data, DST, cold/warming, and solar-trend fixtures.


## Frontend design

The frontend uses MUI with a navy/yellow theme, system fonts, compact surfaces, shared
controls, semantic notifications, and responsive tables. Reusable React/CSS
components live in `components/common`, weather views in `components/weather`,
API requests in `services`, and date formatting in `utils`. Forecast calculations
remain in Go. Fonts are provided by the user's system; no font assets are bundled.

## Languages

Choose English or Suomi in the application bar. The choice is saved in browser
local storage; switching language does not fetch weather again or reset the city
or forecast view. Date labels follow the selected language while preserving
provider-local forecast times.

Shared text lives in `frontend/src/i18n/locales/en/weather.json` and `fi/weather.json`. Each file contains a `weather` namespace with descriptive snake_case keys
(for example, `weather.full_day`). Keep keys stable when editing values. Components
use `useI18n().t()`; Finnish entries are checked against the English keys by
TypeScript. Add or edit translations there rather than in individual components.
Placeholders such as `{count}` keep changing values separate from translations.

The weather API remains English. `frontend/src/i18n/signals.ts` adapts known
insight message templates for display without repeating weather calculations.
If backend wording changes, update the matching locale template; unknown messages
remain in English. A future API could return stable message keys and parameters
to remove that coupling. Units and numerical weather values are unchanged.

Run `cd frontend && npm test` (Node 22.13+), then `npm run build`. Translation
tests check locale coverage, placeholders, dynamic insights, and unknown-message
fallback. Manually switch languages in both forecast views, including loading,
retry, tooltips, table coverage, and insights.

MUI components provide the dropdown menus, controls, tooltips, table, alerts,
application bar, and surfaces. Shared colors, typography, and component defaults
are defined in `frontend/src/theme.ts`; CSS retains the responsive weather layout
and Recharts styling. Recharts still owns all chart rendering.

## Wind generation potential

Wind conditions help electricity retailers monitor weather-driven supply
uncertainty alongside temperature-related demand and solar availability.
The existing 14-day Open-Meteo call also requests `wind_speed_120m` and
`wind_direction_120m`, with `wind_speed_unit=ms`. See the
[Open-Meteo forecast documentation](https://open-meteo.com/en/docs).
120 m is a useful illustrative hub-height proxy, closer to turbine operating
height than near-surface 10 m wind; actual hub heights and local terrain vary.
Direction is shown in degrees indicating where the wind comes from.

Hourly wind tooltips show the backend-calculated qualitative categories: below 3 m/s very low; 3 to below
6 low; 6 to below 10 moderate; 10 through 15 high; above 15 strong wind with
uncertain turbine-specific output. These are illustrative heuristics, not
validated turbine operating ranges or a power curve. Strong winds may lead to
output reduction or safety shutdown; more wind does not always mean more output.
No MW/MWh, actual generation, demand, or price forecast is calculated. Installed
capacity, turbine curves, availability, terrain, and regional generation data
would be required for a production forecast.

The three daily cards show mean wind at 120 m. The 72-hour detail section
includes an hourly wind chart with direction and potential in its tooltip. The
14-day table and wind chart show daily minimum/minimum/mean/maximum wind; there is no standalone
wind summary card. Chart tooltips mark partial daily coverage. A concise accessible information tooltip explains the qualitative proxy. Wind classification applies to individual hourly speeds, not daily mean
speeds. Missing/null or invalid wind readings remain unavailable, and charts do
not connect gaps. Zero wind is valid. Daily aggregates use available samples and
show partial coverage; missing direction does not discard a valid speed. Direction
is not averaged across days because angles require circular statistics.

To check manually, select a city and inspect the compact wind summaries in the
three daily cards. Hover the hourly wind chart to see speed, direction, and
potential, then switch to the 14-day table and inspect minimum/mean/maximum wind and
coverage. Switch languages and use keyboard focus on the information tooltips. Test fixtures cover boundaries,
nulls, short arrays, zero wind, JSON, and aggregation without calling the live API.

## Daily conditions and forecast labels

“Next 3 days” contains a daily weather summary and hourly weather details.
Cards use daily average/minimum/maximum temperatures, accumulated irradiation,
and mean wind at 120 m. The 14-day outlook retains the daily table and charts.
The sun and wind symbols identify irradiation and wind metrics; they do not
represent weather conditions. No weather-condition code is requested.

Daily summary cards also show the circular mean wind-from direction at 120 m. The arrow points toward the source (north means wind from the north), alongside an eight-point compass label. The mean gives equal weight to available non-calm hourly bearings; missing readings are excluded and cancelling bearings leave the direction unavailable. This daily mean can hide changes within the day; use hourly wind tooltips for individual measurements.

### Wind generation insights

The existing `signals` and `outlookSignals` arrays now include `kind: "wind"`, rendered in the existing trading insight cards. Wind speed remains a qualitative proxy at 120 m, not a forecast of MW, MWh, demand, prices or grid shortages. The daily API also exposes `windDirectionConsistency` (mean resultant length, 0–1).

The next-3-days detector uses only the upcoming 72 hours, grouped into complete, consecutive six-hour windows in the forecast location's timezone. Speed trends require at least three windows (18 hours), two actual changes, monotonic window means and a significant total change. Missing speeds or timestamp gaps break a window. A low-wind period requires every hourly reading to be below the threshold, not just its window mean.

The 14-day detector reuses the existing `trendPeriod` helper on daily means: at least three consecutive complete days, two actual changes, monotonic means and a significant overall change. Incomplete days or missing wind readings break a trend. Low daily mean wind must persist for at least two complete consecutive days; an individual day may still contain higher hourly speeds. The strongest qualifying speed trend in each direction (weakening and improving), one longest low-wind period and one largest qualifying direction shift are reported per view. Opposite speed trends describe their own dated periods.

| Environment variable | Default | Meaning |
| --- | --- | --- |
| `WIND_SPEED_CHANGE_MS` | 3 | Minimum sustained mean-speed change, m/s |
| `WIND_LOW_SPEED_MS` | 3 | Low-wind threshold; readings/means must be strictly below, m/s |
| `WIND_MIN_LOW_HOURS` | 12 | Minimum consecutive low-wind hours (12–72; rounded up to complete six-hour windows) |
| `WIND_MIN_LOW_DAYS` | 2 | Minimum consecutive complete low-wind days (2–14) |
| `WIND_DIRECTION_CHANGE_DEG` | 60 | Minimum circular direction shift, degrees (up to 180) |
| `WIND_DIRECTION_CONSISTENCY` | 0.8 | Minimum directional mean resultant length (up to 1) |
| `WIND_STRONG_SPEED_MS` | 15 | Suppress speed-trend generation headlines if any included hourly reading exceeds this value |
| `WIND_COMBINED_LOW_SOLAR_KWH_M2` | 1 | Maximum daily solar-energy proxy for a combined warning, kWh/m² |
| `WIND_COMBINED_LOW_SOLAR_W_M2` | 100 | Maximum six-hour mean solar irradiance for a combined warning, W/m² |

Direction shifts compare adjacent six-hour windows or complete days, requiring every direction reading, non-calm winds, mean speeds at least the low-wind threshold, and consistent bearings in both periods. Circular differences correctly treat 350° to 10° as 20°, not 340°. Bearings describe where wind comes from; variable/opposing or calm winds do not produce a direction-change insight.

A combined warning requires a sustained low-wind period with complete temperature and solar data throughout, temperature falling at least 3°C and ending below the existing 18°C heating reference, and solar availability below its configured limit throughout. It replaces an existing combined card in that view to avoid duplicate or opposing combined headlines. Existing temperature and solar detectors are unchanged. Empty frontend insight categories are hidden.

These thresholds are illustrative, configurable heuristics, not calibrated turbine operating ranges. Wind direction effects depend on site and turbine characteristics; high winds can lead to output reduction or shutdown. Six-hour averaging can hide shorter changes, and daily averaging can hide hourly variability. Low hourly solar irradiance includes nighttime, so a short-term combined warning describes its specified period rather than a full day's PV outlook. Missing/partial data can suppress insights; silence is not evidence of stable generation or prices. DST ambiguities or gaps conservatively break hourly runs.

To check the feature manually, fetch a forecast, open both forecast views, and inspect the Wind generation conditions category when a qualifying period exists. No wind card appears when there is no meaningful signal. Change environment thresholds and restart the backend (clearing its in-memory cache) to evaluate different heuristics. Automated tests use controlled forecasts; production still uses Open-Meteo.
