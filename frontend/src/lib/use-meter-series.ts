import { useQuery } from "@tanstack/react-query"
import { useMemo } from "react"

import { api } from "@/lib/api"
import { buildChartRows, dayTicks } from "@/lib/chart-data"

export function useMeterSeries(meterId: string | undefined) {
  const enabled = Boolean(meterId)
  const readings = useQuery({
    queryKey: ["meter-readings", meterId],
    queryFn: () => api.meterReadings(meterId!),
    enabled,
  })
  const baseline = useQuery({
    queryKey: ["meter-baseline", meterId],
    queryFn: () => api.meterBaseline(meterId!),
    enabled,
  })
  const events = useQuery({
    queryKey: ["events", meterId],
    queryFn: () => api.events(meterId!),
    enabled,
  })

  const rows = useMemo(
    () =>
      readings.data && baseline.data
        ? buildChartRows(readings.data, baseline.data.hours)
        : [],
    [readings.data, baseline.data]
  )
  const ticks = useMemo(() => dayTicks(rows), [rows])

  return {
    rows,
    ticks,
    events: events.data ?? [],
    error: readings.error ?? baseline.error ?? events.error,
  }
}
