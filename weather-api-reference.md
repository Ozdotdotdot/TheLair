# Weather API Reference

Two APIs working together: OpenWeatherMap for live current conditions (with thunderstorm support),
Open-Meteo as a no-key fallback cache.

---

## Architecture

```
On boot + daily at midnight:
  → Fetch Open-Meteo hourly forecast (no key required)
  → Cache to disk: { hour: weather_code } for next 24 hours

Every 30 minutes (primary loop):
  → Call OpenWeatherMap current conditions
  → If success:  use live OWM data → update display
  → If failure:  read cache[current_hour] → use Open-Meteo fallback
  → If both fail: breathing idle state
```

Why this split:
- OWM returns thunderstorm codes for the US. Open-Meteo does not (US limitation).
- Open-Meteo requires no API key, making it a reliable background safety net.
- OWM free tier allows ~1M calls/month. 48 calls/day = 0.14% of quota.
- If internet drops, the keyboard still shows a meaningful weather state.

---

## Display States

Six states cover everything relevant for Alpharetta, GA. Drizzle folds into rain,
fog/haze folds into cloudy — neither is common enough to warrant a dedicated state.

| State | OWM codes | Open-Meteo codes | Real world |
|---|---|---|---|
| Clear | 800 | 0, 1 | Sunny, no clouds |
| Cloudy | 801–804 | 2, 3, 45, 48 | Overcast, fog, haze |
| Rain | 3xx, 500–504, 511 | 51–57, 61–67 | Steady frontal rain, drizzle, freezing rain |
| Showers | 520–531 | 80–82, 85–86 | Brief convective cells, afternoon storms |
| Snow | 6xx | 71–77 | Any snowfall |
| Thunderstorm | 2xx | *(not available for US)* | Lightning, active storm |

> For the fallback (Open-Meteo), thunderstorms in the US will appear as heavy showers (82).
> That's acceptable degraded behavior — the live OWM call is what catches real thunderstorms.

---

## OpenWeatherMap

**Base URL:** `https://api.openweathermap.org/data/2.5/weather`
**Requires:** Free API key (1M calls/month free tier)
**Use:** Live current conditions, polled every 30 minutes

### Current Conditions Query

```
GET https://api.openweathermap.org/data/2.5/weather
  ?lat=34.0518
  &lon=-84.2742
  &units=imperial
  &appid={API_KEY}
```

### Response Structure

```json
{
  "weather": [
    {
      "id": 211,
      "main": "Thunderstorm",
      "description": "thunderstorm",
      "icon": "11d"
    }
  ],
  "main": {
    "temp": 72.3,
    "feels_like": 70.1,
    "humidity": 65
  },
  "dt": 1713196800
}
```

The `weather[0].id` integer is what you parse for display state.
The `icon` suffix encodes day (`d`) or night (`n`) — e.g. `11d` vs `11n`.

### Code Structure

OWM uses a 3-digit grouped system. The first digit gives you the category:

```
id / 100 == 2  →  Thunderstorm
id / 100 == 3  →  Drizzle
id / 100 == 5  →  Rain
id / 100 == 6  →  Snow
id / 100 == 7  →  Atmosphere (fog, mist, haze, tornado)
id == 800      →  Clear
id / 100 == 8  →  Clouds  (801–804)
```

A single integer divide is enough to route to a display state.

### Complete Code Table

**Thunderstorm (2xx)**

| Code | Description |
|---|---|
| 200 | Thunderstorm with light rain |
| 201 | Thunderstorm with rain |
| 202 | Thunderstorm with heavy rain |
| 210 | Light thunderstorm |
| 211 | Thunderstorm |
| 212 | Heavy thunderstorm |
| 221 | Ragged thunderstorm (fast-moving squall) |
| 230 | Thunderstorm with light drizzle |
| 231 | Thunderstorm with drizzle |
| 232 | Thunderstorm with heavy drizzle |

**Drizzle (3xx)** → maps to Rain state

| Code | Description |
|---|---|
| 300 | Light intensity drizzle |
| 301 | Drizzle |
| 302 | Heavy intensity drizzle |
| 310 | Light intensity drizzle rain |
| 311 | Drizzle rain |
| 312 | Heavy intensity drizzle rain |
| 313 | Shower rain and drizzle |
| 314 | Heavy shower rain and drizzle |
| 321 | Shower drizzle |

**Rain (5xx)**

| Code | Description | State |
|---|---|---|
| 500 | Light rain | Rain |
| 501 | Moderate rain | Rain |
| 502 | Heavy intensity rain | Rain |
| 503 | Very heavy rain | Rain |
| 504 | Extreme rain | Rain |
| 511 | Freezing rain | Rain |
| 520 | Light intensity shower rain | Showers |
| 521 | Shower rain | Showers |
| 522 | Heavy intensity shower rain | Showers |
| 531 | Ragged shower rain | Showers |

**Snow (6xx)**

| Code | Description |
|---|---|
| 600 | Light snow |
| 601 | Snow |
| 602 | Heavy snow |
| 611 | Sleet |
| 612 | Light shower sleet |
| 613 | Shower sleet |
| 615 | Light rain and snow |
| 616 | Rain and snow |
| 620 | Light shower snow |
| 621 | Shower snow |
| 622 | Heavy shower snow |

**Atmosphere (7xx)** → maps to Cloudy state

| Code | Description |
|---|---|
| 701 | Mist |
| 711 | Smoke |
| 721 | Haze |
| 731 | Dust/sand whirls |
| 741 | Fog |
| 751 | Sand |
| 761 | Dust |
| 771 | Squalls |
| 781 | Tornado |

**Clear / Clouds (800–804)**

| Code | Description | State |
|---|---|---|
| 800 | Clear sky | Clear |
| 801 | Few clouds (11–25%) | Cloudy |
| 802 | Scattered clouds (25–50%) | Cloudy |
| 803 | Broken clouds (51–84%) | Cloudy |
| 804 | Overcast clouds (85–100%) | Cloudy |

---

## Open-Meteo (Fallback Cache)

**Base URL:** `https://api.open-meteo.com/v1/forecast`
**Requires:** Nothing — no API key, no account
**Use:** Fetched once daily, cached to disk, used only if OWM call fails

### Daily Cache Query

```
GET https://api.open-meteo.com/v1/forecast
  ?latitude=34.0518
  &longitude=-84.2742
  &hourly=weather_code
  &daily=sunrise,sunset
  &timezone=America/New_York
  &forecast_days=1
```

This returns 24 hourly weather codes for today plus sunrise/sunset timestamps.
Cache this response. On OWM failure, read `hourly.weather_code[current_hour]`.

### Query Parameters

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `latitude` | float | Yes | — | WGS84 latitude |
| `longitude` | float | Yes | — | WGS84 longitude |
| `current` | string array | No | — | Variables for current snapshot |
| `hourly` | string array | No | — | Variables as hourly forecast |
| `daily` | string array | No | — | Daily aggregates (requires `timezone`) |
| `timezone` | string | No | GMT | IANA timezone or `auto` |
| `temperature_unit` | string | No | `celsius` | `fahrenheit` |
| `wind_speed_unit` | string | No | `kmh` | `ms`, `mph`, `kn` |
| `precipitation_unit` | string | No | `mm` | `inch` |
| `timeformat` | string | No | `iso8601` | `unixtime` |
| `forecast_days` | int 0–16 | No | 7 | Days of forecast |
| `past_days` | int 0–92 | No | 0 | Days of history to include |
| `forecast_hours` | int | No | — | Limit hourly timesteps |
| `start_date` / `end_date` | string yyyy-mm-dd | No | — | Explicit date range |
| `elevation` | float | No | — | Override elevation; `nan` disables |
| `cell_selection` | string | No | `land` | `sea`, `nearest` |
| `models` | string array | No | `auto` | Force specific forecast model |

### Useful Variables

| Variable | Use in |
|---|---|
| `weather_code` | `current`, `hourly`, `daily` |
| `temperature_2m` | `current`, `hourly` |
| `precipitation` | `current`, `hourly` |
| `cloud_cover` | `current`, `hourly` |
| `is_day` | `current`, `hourly` — 1 if daytime, 0 if night |
| `sunrise` / `sunset` | `daily` |
| `wind_speed_10m` | `current`, `hourly` |
| `visibility` | `current`, `hourly` |

### WMO Code System

Open-Meteo uses WMO (World Meteorological Organization) numeric codes. They are organized
into bands — like HTTP status codes — where the range indicates the category and gaps are
intentional forward-compatibility space.

**Band overview**

| Range | Category |
|---|---|
| 0–3 | Sky cover (clear → overcast) |
| 45, 48 | Fog |
| 51–57 | Drizzle |
| 61–67 | Rain |
| 71–77 | Snow |
| 80–82 | Rain showers |
| 85–86 | Snow showers |
| 95–99 | Thunderstorm (Central Europe only — not available for US) |

**Why odd numbers for intensity?**

Intensity within a band steps at +0 / +2 / +4 from the base:
```
61 = rain (slight)
63 = rain (moderate)
65 = rain (heavy)
```
Gaps (62, 64) are reserved for future WMO additions without breaking existing mappings.

**Complete code table**

| Code | Condition |
|---|---|
| 0 | Clear sky |
| 1 | Mainly clear |
| 2 | Partly cloudy |
| 3 | Overcast |
| | |
| 45 | Fog |
| 48 | Depositing rime fog |
| | |
| 51 | Drizzle — light |
| 53 | Drizzle — moderate |
| 55 | Drizzle — dense |
| 56 | Freezing drizzle — light |
| 57 | Freezing drizzle — dense |
| | |
| 61 | Rain — slight |
| 63 | Rain — moderate |
| 65 | Rain — heavy |
| 66 | Freezing rain — light |
| 67 | Freezing rain — heavy |
| | |
| 71 | Snowfall — slight |
| 73 | Snowfall — moderate |
| 75 | Snowfall — heavy |
| 77 | Snow grains |
| | |
| 80 | Rain showers — slight |
| 81 | Rain showers — moderate |
| 82 | Rain showers — violent |
| 85 | Snow showers — slight |
| 86 | Snow showers — heavy |
| | |
| 95 | Thunderstorm — slight or moderate *(Central Europe only)* |
| 96 | Thunderstorm with slight hail *(Central Europe only)* |
| 99 | Thunderstorm with heavy hail *(Central Europe only)* |

**Rain vs. Showers**

- **Rain (61–67):** Continuous, frontal precipitation. Wide area, lasts hours. Steady grey sky.
- **Showers (80–82):** Convective cells — brief, intense, localized. Sun can return minutes later.
  Classic Georgia afternoon storm pattern.
