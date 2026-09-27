import {
  Area,
  CartesianGrid,
  ComposedChart,
  Line,
  ReferenceArea,
  ReferenceLine,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"

import { ChartContainer, type ChartConfig } from "@/components/ui/chart"
import type { ChartRow } from "@/lib/chart-data"
import { formatNumber, formatPlantDateTime } from "@/lib/format"
import type { OperationalEvent } from "@/lib/types"

const COLOR = {
  series: "var(--series-1)",
  band: "var(--chart-1)",
  baseline: "var(--chart-2)",
  alert: "var(--destructive)",
}

function alertDot(isMarked: (row: ChartRow) => boolean, radius: number) {
  return (props: {
    cx?: number
    cy?: number
    index?: number
    payload?: ChartRow
  }) => {
    const { cx, cy, index, payload } = props
    if (
      !payload ||
      !isMarked(payload) ||
      cx === undefined ||
      cy === undefined
    ) {
      return <g key={index} />
    }
    return <circle key={index} cx={cx} cy={cy} r={radius} fill={COLOR.alert} />
  }
}

const dayFormat = new Intl.DateTimeFormat("es-CO", {
  timeZone: "UTC",
  day: "2-digit",
  month: "2-digit",
})
const formatDay = (t: number) => dayFormat.format(new Date(t))

const axisProps = {
  tickLine: false,
  axisLine: false,
  tickMargin: 8,
  fontSize: 11,
} as const

interface TooltipRow {
  label: string
  value: string
  color?: string
}

function TooltipBox({ t, rows }: { t: number; rows: TooltipRow[] }) {
  return (
    <div className="grid min-w-44 gap-1.5 rounded-lg border bg-background px-2.5 py-1.5 text-xs shadow-xl">
      <div className="font-medium">
        {formatPlantDateTime(new Date(t).toISOString())}
      </div>
      {rows.map((row) => (
        <div
          key={row.label}
          className="flex items-center justify-between gap-4"
        >
          <span className="flex items-center gap-1.5 text-muted-foreground">
            {row.color && (
              <span
                className="size-2 rounded-[2px]"
                style={{ background: row.color }}
              />
            )}
            {row.label}
          </span>
          <span className="font-mono font-medium text-foreground tabular-nums">
            {row.value}
          </span>
        </div>
      ))}
    </div>
  )
}

type TooltipArgs = {
  active?: boolean
  payload?: ReadonlyArray<{ payload?: unknown }>
}
const rowFrom = ({ active, payload }: TooltipArgs) =>
  active ? (payload?.[0]?.payload as ChartRow | undefined) : undefined

const consumptionConfig = {
  kwh: { label: "Consumo", color: COLOR.series },
  band: { label: "Rango normal", color: COLOR.band },
  outOfBand: { label: "Fuera del rango", color: COLOR.alert },
} satisfies ChartConfig

export function ConsumptionChart({
  rows,
  ticks,
  events,
  detection,
  thresholdPct,
}: {
  rows: ChartRow[]
  ticks: number[]
  events: OperationalEvent[]
  detection?: { start: number; end: number; label: string }
  thresholdPct?: number // umbral del motor (GET /analysis/rules)
}) {
  const outOfBand = rows.filter((r) => r.outOfBand)

  return (
    <div className="flex flex-col gap-3">
      <Legend
        items={[
          { label: consumptionConfig.kwh.label, swatch: COLOR.series },
          {
            label: thresholdPct
              ? `${consumptionConfig.band.label} (baseline ±${thresholdPct} %)`
              : consumptionConfig.band.label,
            swatch: COLOR.band,
            area: true,
          },
          ...(outOfBand.length > 0
            ? [
                {
                  label: `${consumptionConfig.outOfBand.label} (${outOfBand.length} h)`,
                  swatch: COLOR.alert,
                  dot: true,
                },
              ]
            : []),
        ]}
      />
      <ChartContainer
        config={consumptionConfig}
        className="aspect-auto h-72 w-full"
      >
        <ComposedChart
          data={rows}
          margin={{ top: 16, right: 8, left: 0, bottom: 0 }}
        >
          <CartesianGrid vertical={false} />
          <XAxis
            dataKey="t"
            type="number"
            scale="time"
            domain={["dataMin", "dataMax"]}
            ticks={ticks}
            tickFormatter={formatDay}
            {...axisProps}
          />
          <YAxis
            width={44}
            {...axisProps}
            tickFormatter={(v: number) => formatNumber(v, 0)}
            unit=""
          />

          {detection && (
            <ReferenceArea
              x1={detection.start}
              x2={detection.end}
              fill={COLOR.alert}
              fillOpacity={0.07}
              label={{
                value: detection.label,
                position: "insideBottomLeft",
                fontSize: 11,
                fill: "var(--muted-foreground)",
              }}
            />
          )}

          <Area
            dataKey="band"
            type="monotone"
            stroke="none"
            fill={COLOR.band}
            fillOpacity={0.35}
            isAnimationActive={false}
            activeDot={false}
          />
          <Line
            dataKey="kwh"
            type="monotone"
            stroke={COLOR.series}
            strokeWidth={2}
            dot={alertDot((row) => row.outOfBand, 2.5)}
            activeDot={{ r: 4 }}
            isAnimationActive={false}
          />

          {events.map((event) => (
            <ReferenceLine
              key={event.id}
              x={Date.parse(event.event_timestamp)}
              stroke="var(--foreground)"
              strokeOpacity={0.6}
              label={{
                value: `⚑ ${event.event_type}`,
                position: "insideTopLeft",
                fontSize: 10,
                fill: "var(--foreground)",
              }}
            />
          ))}

          <Tooltip
            cursor={{ stroke: "var(--muted-foreground)", strokeWidth: 1 }}
            content={(args: TooltipArgs) => {
              const row = rowFrom(args)
              if (!row) return null
              const deviation = row.baselineKwh
                ? (row.kwh / row.baselineKwh - 1) * 100
                : null
              return (
                <TooltipBox
                  t={row.t}
                  rows={[
                    {
                      label: "Consumo",
                      value: `${formatNumber(row.kwh, 2)} kWh`,
                      color: COLOR.series,
                    },
                    ...(row.band
                      ? [
                          {
                            label: "Rango normal",
                            value: `${formatNumber(row.band[0], 1)}–${formatNumber(row.band[1], 1)}`,
                            color: COLOR.band,
                          },
                        ]
                      : []),
                    ...(deviation !== null
                      ? [
                          {
                            label: "Desviación",
                            value: `${deviation > 0 ? "+" : ""}${formatNumber(deviation, 1)} %`,
                            color: row.outOfBand ? COLOR.alert : undefined,
                          },
                        ]
                      : []),
                  ]}
                />
              )
            }}
          />
        </ComposedChart>
      </ChartContainer>
    </div>
  )
}

type MetricKey = "voltage" | "current" | "pf"

export function MetricChart({
  rows,
  ticks,
  metric,
  unit,
  decimals,
  baselineKey,
  validRange,
}: {
  rows: ChartRow[]
  ticks: number[]
  metric: MetricKey
  unit: string
  decimals: number
  baselineKey?: "baselineCurrent" | "baselinePf"
  validRange?: [number, number]
}) {
  const config = {
    [metric]: { label: "Medido", color: COLOR.series },
    baseline: { label: "Baseline", color: COLOR.baseline },
    invalid: { label: "Fuera de rango", color: COLOR.alert },
  } satisfies ChartConfig
  const isInvalid = (r: ChartRow) =>
    validRange !== undefined &&
    (r[metric] < validRange[0] || r[metric] > validRange[1])
  const invalidCount = rows.filter(isInvalid).length

  return (
    <div className="flex flex-col gap-2">
      <Legend
        items={[
          { label: "Medido", swatch: COLOR.series },
          ...(baselineKey
            ? [
                {
                  label: "Baseline de su hora",
                  swatch: COLOR.baseline,
                },
              ]
            : []),
          ...(validRange
            ? [
                {
                  label: `Rango válido ${validRange[0]}–${validRange[1]} ${unit}`,
                  swatch: COLOR.band,
                  area: true,
                },
              ]
            : []),
          ...(invalidCount > 0
            ? [
                {
                  label: `Fuera de rango (${invalidCount})`,
                  swatch: COLOR.alert,
                  dot: true,
                },
              ]
            : []),
        ]}
      />
      <ChartContainer config={config} className="aspect-auto h-44 w-full">
        <ComposedChart
          data={rows}
          margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
        >
          <CartesianGrid vertical={false} />
          <XAxis
            dataKey="t"
            type="number"
            scale="time"
            domain={["dataMin", "dataMax"]}
            ticks={ticks.filter((_, i) => i % 2 === 0)}
            tickFormatter={formatDay}
            {...axisProps}
          />
          <YAxis
            width={48}
            domain={["auto", "auto"]}
            {...axisProps}
            tickFormatter={(v: number) =>
              formatNumber(v, decimals >= 3 ? 2 : 0)
            }
          />
          {validRange && (
            <ReferenceArea
              ifOverflow="extendDomain"
              y1={validRange[0]}
              y2={validRange[1]}
              fill={COLOR.band}
              fillOpacity={0.35}
            />
          )}
          {baselineKey && (
            <Line
              dataKey={baselineKey}
              type="monotone"
              stroke={COLOR.baseline}
              strokeWidth={1.5}
              dot={false}
              isAnimationActive={false}
            />
          )}
          <Line
            dataKey={metric}
            type="monotone"
            stroke={COLOR.series}
            strokeWidth={2}
            dot={invalidCount > 0 ? alertDot(isInvalid, 3) : false}
            activeDot={{ r: 4 }}
            isAnimationActive={false}
          />
          <Tooltip
            cursor={{ stroke: "var(--muted-foreground)", strokeWidth: 1 }}
            content={(args: TooltipArgs) => {
              const row = rowFrom(args)
              if (!row) return null
              const base = baselineKey ? row[baselineKey] : null
              return (
                <TooltipBox
                  t={row.t}
                  rows={[
                    {
                      label: "Medido",
                      value: `${formatNumber(row[metric], decimals)} ${unit}`,
                      color: COLOR.series,
                    },
                    ...(base !== null
                      ? [
                          {
                            label: "Baseline",
                            value: `${formatNumber(base, decimals)} ${unit}`,
                            color: COLOR.baseline,
                          },
                        ]
                      : []),
                  ]}
                />
              )
            }}
          />
        </ComposedChart>
      </ChartContainer>
    </div>
  )
}

function Legend({
  items,
}: {
  items: { label: string; swatch: string; area?: boolean; dot?: boolean }[]
}) {
  return (
    <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
      {items.map((item) => (
        <span key={item.label} className="flex items-center gap-1.5">
          <span
            className={
              item.dot
                ? "size-2 rounded-full"
                : item.area
                  ? "h-2.5 w-3 rounded-[2px]"
                  : "h-0.5 w-3 rounded-full"
            }
            style={{ background: item.swatch, opacity: item.area ? 0.6 : 1 }}
          />
          {item.label}
        </span>
      ))}
    </div>
  )
}
