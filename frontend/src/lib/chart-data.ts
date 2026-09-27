import type { HourBaseline, Reading } from "@/lib/types"

export interface ChartRow {
  t: number
  kwh: number
  voltage: number
  current: number
  pf: number
  baselineKwh: number | null
  band: [number, number] | null
  baselineCurrent: number | null
  baselinePf: number | null
  outOfBand: boolean
}

export function buildChartRows(
  readings: Reading[],
  baseline: HourBaseline[]
): ChartRow[] {
  const byHour = new Map(baseline.map((b) => [b.hour, b]))

  return readings.map((r) => {
    const t = Date.parse(r.timestamp)
    const b = byHour.get(new Date(t).getUTCHours())
    return {
      t,
      kwh: r.consumption_kwh,
      voltage: r.voltage_v,
      current: r.current_a,
      pf: r.power_factor,
      baselineKwh: b?.kwh ?? null,
      band: b ? [b.lower_kwh, b.upper_kwh] : null,
      baselineCurrent: b?.current_a ?? null,
      baselinePf: b?.power_factor ?? null,
      outOfBand: b
        ? r.consumption_kwh < b.lower_kwh || r.consumption_kwh > b.upper_kwh
        : false,
    }
  })
}

export function dayTicks(rows: ChartRow[]) {
  if (rows.length === 0) return []
  const DAY = 24 * 60 * 60 * 1000
  const first = Math.ceil(rows[0].t / DAY) * DAY
  const last = rows[rows.length - 1].t
  const ticks: number[] = []
  for (let t = first; t <= last; t += DAY) ticks.push(t)
  return ticks
}
