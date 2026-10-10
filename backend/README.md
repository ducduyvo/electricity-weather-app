# Go backend

The server wires dependencies explicitly:

`GET /api/weather → Handler → Provider → Cached → OpenMeteo → external API`

The weather package keeps models, HTTP handling, upstream mapping, caching,
validation, summaries, and signals in separate files. `Provider` remains a
single-method interface. There is no forwarding service layer or persistence.
`GET /api/health` returns the health status.

## Configuration

| Environment variable | Default |
| --- | --- |
| `SERVER_ADDRESS` | `:8080` |
| `OPEN_METEO_BASE_URL` | `https://api.open-meteo.com/v1/forecast` |
| `HTTP_CLIENT_TIMEOUT` | `5s` |
| `CACHE_TTL` | `10m` |
| `ALLOWED_FRONTEND_ORIGIN` | `*` |

Durations must be positive Go durations. The provider URL must be an absolute
HTTP or HTTPS URL. Defaults preserve the original local setup. Set a specific
frontend origin when deploying.

The server uses read/header/idle/write timeouts. SIGINT and SIGTERM initiate
shutdown with a five-second deadline; a second signal can terminate immediately.

## Cache and calculations

Cache keys round latitude and longitude to three decimal places. Nearby points
can share a cached forecast, including its original coordinates. Upstream queries
use four decimal places. Successful results are cached for the TTL; expired
entries are removed lazily on subsequent requests. The cache does not hold its
mutex during network calls. Concurrent misses can issue duplicate requests.
Request coalescing or a capacity bound could be future improvements; there is
no background refresh or distributed cache. Cached forecast slices are treated
as read-only by consumers.

Daily temperature averages/min/max ignore null readings. Positive hourly
shortwave radiation is summed and divided by 1000 as a solar-energy proxy,
assuming one-hour intervals. Signals retain the existing inclusive thresholds:
±3°C average temperature change and ±0.5 kWh/m² solar-energy change. Temperature trend signals now qualify heating/cooling implications using
18°C/22°C descriptive reference bands. A 500 W/m² peak is labeled a strong
solar period; this is a heuristic, not an estimate of PV electricity output.
Daily summaries add peakSolarIrradiance (nullable), peakSolarTime (local),
solarSampleCount, and hourCount without changing existing fields. Solar
comparisons require complete non-negative hourly readings (including a zero baseline). Peak insights can use partial readings with an explicit caveat.
Missing comparisons produce a limited-data signal; the stable-weather fallback
requires both comparisons. Combined signals remain qualitative and make no
claims about actual demand, generation, or prices.

The unused overlapping internal/insight implementation was removed; all
business calculations remain in the weather package.

Weather endpoint errors use JSON with safe public messages. Upstream errors
retain wrapped causes for logging. Coordinate validation rejects malformed,
out-of-range, NaN, and infinite inputs. GET routes also accept HEAD through
Go's standard ServeMux behavior; other methods receive its standard 405 response.

## Run and verify

From this directory:

```sh
go run ./cmd/server
go test ./...
go test -race ./...
```

Tests use handwritten providers, local HTTP servers, deterministic cache expiry,
and pure calculation fixtures. They do not call the live Open-Meteo API.


## Seven-day outlook

One `forecast_days=7` upstream request supplies both resolutions. `daily`
contains local-calendar summaries and adds `expectedHourCount`, `completeDay`,
and `temperatureSampleCount`. Local IANA timezone day length handles DST;
timezone data is embedded for container runtimes. Invalid timestamps are skipped.
Existing summaries/short-term signals are reused. `outlookSignals` contains
separate multi-day temperature and solar trends over consecutive complete days;
missing values interrupt a trend. Partial daily totals remain identifiable and
are not promoted to full-day forecasts. The frontend shows hourly detail for
72 hours and at most seven aggregated daily entries for the broader outlook.
