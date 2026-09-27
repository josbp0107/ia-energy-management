import { useQuery } from "@tanstack/react-query"
import { ArrowRightIcon, SparklesIcon } from "lucide-react"
import { Link, useSearchParams } from "react-router"

import {
  AnomalyStatusBadge,
  AnomalyTypeBadge,
  SeverityBadge,
} from "@/components/status-badges"
import { ErrorAlert, PageHeader } from "@/components/page"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { api } from "@/lib/api"
import { formatConfidence, formatNumber, formatPct } from "@/lib/format"
import { anomalyStatusLabel } from "@/lib/labels"
import type { Anomaly, AnomalyStatus } from "@/lib/types"
import { cn } from "@/lib/utils"

const STATUS_FILTERS: ("ALL" | AnomalyStatus)[] = [
  "ALL",
  "OPEN",
  "INVESTIGATING",
  "RESOLVED",
  "DISMISSED",
]
type StatusFilter = (typeof STATUS_FILTERS)[number]

const isStatusFilter = (v: string | null): v is StatusFilter =>
  v !== null && (STATUS_FILTERS as string[]).includes(v)

export function AnomaliesPage() {
  const anomalies = useQuery({
    queryKey: ["anomalies"],
    queryFn: () => api.anomalies(),
  })
  const meters = useQuery({ queryKey: ["meters"], queryFn: api.meters })
  const meterName = new Map(meters.data?.map((m) => [m.meter_id, m.name]))

  const [params, setParams] = useSearchParams()
  const raw = params.get("status")
  const filter: StatusFilter = isStatusFilter(raw) ? raw : "ALL"

  const all = anomalies.data ?? []
  const visible =
    filter === "ALL" ? all : all.filter((a) => a.status === filter)
  const count = (f: StatusFilter) =>
    f === "ALL" ? all.length : all.filter((a) => a.status === f).length

  return (
    <>
      <PageHeader
        title="Anomalías IA"
        description="Resultado del último análisis, ordenado por prioridad. Abre una anomalía para ver la evidencia y decidir qué hacer."
      />

      {anomalies.error && (
        <ErrorAlert
          title="No se pudieron cargar las anomalías"
          error={anomalies.error}
        />
      )}

      <ToggleGroup
        type="single"
        variant="outline"
        value={filter}
        onValueChange={(value) =>
          value &&
          setParams(value === "ALL" ? {} : { status: value }, {
            replace: true,
          })
        }
        className="flex-wrap"
      >
        {STATUS_FILTERS.map((f) => (
          <ToggleGroupItem key={f} value={f} className="gap-1.5 px-3">
            {f === "ALL" ? "Todas" : anomalyStatusLabel[f]}
            <span className="text-xs text-muted-foreground tabular-nums">
              {count(f)}
            </span>
          </ToggleGroupItem>
        ))}
      </ToggleGroup>

      <div className="flex flex-col gap-3">
        {anomalies.isPending &&
          Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} className="h-28 w-full" />
          ))}

        {anomalies.isSuccess && all.length === 0 && (
          <Card>
            <CardContent className="flex flex-col items-center gap-2 py-10 text-center">
              <SparklesIcon className="size-8 text-muted-foreground" />
              <p className="font-medium">Todavía no hay análisis</p>
              <p className="text-sm text-muted-foreground">
                Pulsa <span className="font-medium">Run AI Analysis</span> en la
                cabecera para detectar anomalías.
              </p>
            </CardContent>
          </Card>
        )}

        {anomalies.isSuccess && all.length > 0 && visible.length === 0 && (
          <p className="py-6 text-center text-sm text-muted-foreground">
            Ninguna anomalía con este estado.
          </p>
        )}

        {visible.map((anomaly) => (
          <AnomalyCard
            key={anomaly.id}
            anomaly={anomaly}
            rank={all.indexOf(anomaly) + 1}
            meterName={meterName.get(anomaly.meter_id)}
          />
        ))}
      </div>
    </>
  )
}

function AnomalyCard({
  anomaly,
  rank,
  meterName,
}: {
  anomaly: Anomaly
  rank: number
  meterName?: string
}) {
  const dismissedByEngine = !anomaly.anomaly
  return (
    <Link
      to={`/anomalies/${anomaly.id}`}
      className={cn(
        "group block rounded-xl border bg-card transition-colors hover:bg-muted/40",
        dismissedByEngine && "border-dashed opacity-70 hover:opacity-100"
      )}
    >
      <div className="flex gap-4 p-4">
        <div className="flex w-14 shrink-0 flex-col items-center justify-center rounded-lg bg-muted/60 py-2">
          <span className="text-xs text-muted-foreground">#{rank}</span>
          <span className="text-lg font-semibold">
            {formatNumber(anomaly.priority_score, 2)}
          </span>
          <span className="text-[10px] text-muted-foreground">prioridad</span>
        </div>

        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-semibold">{anomaly.meter_id}</span>
            {meterName && (
              <span className="text-sm text-muted-foreground">{meterName}</span>
            )}
            <AnomalyTypeBadge type={anomaly.type} />
            {dismissedByEngine ? (
              <span className="text-xs text-muted-foreground">Sin acción</span>
            ) : (
              <SeverityBadge severity={anomaly.severity} />
            )}
            <span className="ml-auto">
              <AnomalyStatusBadge status={anomaly.status} />
            </span>
          </div>
          <p className="line-clamp-2 text-sm text-muted-foreground">
            {anomaly.reason}
          </p>
          <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
            <span>
              Variación{" "}
              <span className="font-medium text-foreground">
                {formatPct(anomaly.variation_pct, { signed: true })}
              </span>
            </span>
            <span>
              Confianza{" "}
              <span className="font-medium text-foreground">
                {formatConfidence(anomaly.confidence)}
              </span>
            </span>
            <span>
              {anomaly.explanation_source === "llm"
                ? "Explicado por IA"
                : "Explicación por plantilla"}
            </span>
          </div>
        </div>

        <ArrowRightIcon className="size-4 shrink-0 self-center text-muted-foreground transition-transform group-hover:translate-x-0.5" />
      </div>
    </Link>
  )
}
