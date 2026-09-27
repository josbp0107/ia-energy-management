import { useQuery } from "@tanstack/react-query"
import {
  ActivityIcon,
  AlertTriangleIcon,
  ArrowRightIcon,
  BrainCircuitIcon,
  GaugeIcon,
  ShieldCheckIcon,
  SparklesIcon,
  ZapIcon,
  type LucideIcon,
} from "lucide-react"
import { Link } from "react-router"

import {
  AnomalyTypeBadge,
  MeterStatusBadge,
  SeverityBadge,
} from "@/components/status-badges"
import { ErrorAlert, PageHeader } from "@/components/page"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { api } from "@/lib/api"
import {
  formatConfidence,
  formatKWh,
  formatLocalDateTime,
  formatNumber,
  formatPct,
  formatPlantDate,
} from "@/lib/format"
import type { Anomaly, MeterSummary } from "@/lib/types"
import { cn } from "@/lib/utils"

export function DashboardPage() {
  const summary = useQuery({
    queryKey: ["dashboard-summary"],
    queryFn: api.dashboardSummary,
  })
  const anomalies = useQuery({
    queryKey: ["anomalies"],
    queryFn: () => api.anomalies(),
  })
  const meters = useQuery({ queryKey: ["meters"], queryFn: api.meters })

  const error = summary.error ?? anomalies.error ?? meters.error
  const data = summary.data
  const hasAnalysis = Boolean(data?.last_analysis)

  return (
    <>
      <PageHeader
        title="Dashboard"
        description={
          data?.period_start && data.period_end
            ? `Periodo analizado: ${formatPlantDate(data.period_start)} – ${formatPlantDate(data.period_end)}`
            : "Resumen del consumo y de las anomalías detectadas por IA."
        }
      />

      {error && (
        <ErrorAlert title="No se pudieron cargar los datos" error={error} />
      )}

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <KpiCard
          title="Medidores"
          icon={GaugeIcon}
          value={data && formatNumber(data.total_meters, 0)}
          hint="monitoreados"
        />
        <KpiCard
          title="Consumo total"
          icon={ZapIcon}
          value={data && formatKWh(data.total_consumption_kwh, 0)}
          hint="en el periodo"
        />
        <KpiCard
          title="Anomalías IA"
          icon={SparklesIcon}
          value={
            data &&
            (hasAnalysis ? formatNumber(data.anomalies_detected, 0) : "—")
          }
          hint={hasAnalysis ? "en el último análisis" : "sin análisis todavía"}
        />
        <KpiCard
          title="Alta prioridad"
          icon={AlertTriangleIcon}
          value={
            data && (hasAnalysis ? formatNumber(data.high_priority, 0) : "—")
          }
          hint="severidad alta"
          highlight={Boolean(data && data.high_priority > 0)}
        />
        <KpiCard
          title="Confianza agregada"
          icon={BrainCircuitIcon}
          value={
            data && (hasAnalysis ? formatConfidence(data.avg_confidence) : "—")
          }
          hint="promedio de las anomalías"
        />
        <KpiCard
          title="Último análisis"
          icon={ActivityIcon}
          value={
            data &&
            (data.last_analysis
              ? formatLocalDateTime(data.last_analysis.started_at)
              : "Nunca")
          }
          hint={data?.last_analysis ? "completado" : "ejecuta Run AI Analysis"}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <PriorityCard
          className="lg:col-span-2"
          anomalies={anomalies.data}
          isLoading={anomalies.isPending}
        />
        <MeterStatusCard meters={meters.data} isLoading={meters.isPending} />
      </div>
    </>
  )
}

function KpiCard({
  title,
  icon: Icon,
  value,
  hint,
  highlight = false,
}: {
  title: string
  icon: LucideIcon
  value: string | undefined
  hint: string
  highlight?: boolean
}) {
  return (
    <Card className={cn(highlight && "border-destructive/40")}>
      <CardHeader className="flex flex-row items-center justify-between gap-2">
        <CardDescription>{title}</CardDescription>
        <Icon
          className={cn(
            "size-4 text-muted-foreground",
            highlight && "text-destructive"
          )}
        />
      </CardHeader>
      <CardContent>
        {value === undefined ? (
          <Skeleton className="h-8 w-28" />
        ) : (
          <div
            className={cn(
              "text-2xl font-semibold",
              highlight && "text-destructive"
            )}
          >
            {value}
          </div>
        )}
        <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
      </CardContent>
    </Card>
  )
}

function PriorityCard({
  anomalies,
  isLoading,
  className,
}: {
  anomalies: Anomaly[] | undefined
  isLoading: boolean
  className?: string
}) {
  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>Requieren atención</CardTitle>
        <CardDescription>
          Anomalías del último análisis, ordenadas por prioridad.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        {isLoading &&
          Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}

        {anomalies?.length === 0 && (
          <p className="py-6 text-center text-sm text-muted-foreground">
            Aún no se ha ejecutado el análisis de IA.
          </p>
        )}

        {anomalies?.map((anomaly, index) => (
          <Link
            key={anomaly.id}
            to={`/anomalies/${anomaly.id}`}
            className={cn(
              "group flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-muted/50",
              !anomaly.anomaly && "border-dashed opacity-60 hover:opacity-100"
            )}
          >
            <span className="w-5 text-sm font-medium text-muted-foreground tabular-nums">
              {index + 1}
            </span>
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{anomaly.meter_id}</span>
                <AnomalyTypeBadge type={anomaly.type} />
                {anomaly.anomaly ? (
                  <SeverityBadge severity={anomaly.severity} />
                ) : (
                  <span className="text-xs text-muted-foreground">
                    Sin acción
                  </span>
                )}
              </div>
              <p className="truncate text-xs text-muted-foreground">
                Variación {formatPct(anomaly.variation_pct, { signed: true })} ·
                confianza {formatConfidence(anomaly.confidence)}
              </p>
            </div>
            <div className="text-right">
              <div className="text-lg font-semibold tabular-nums">
                {formatNumber(anomaly.priority_score, 2)}
              </div>
              <div className="text-xs text-muted-foreground">prioridad</div>
            </div>
            <ArrowRightIcon className="size-4 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
          </Link>
        ))}
      </CardContent>
    </Card>
  )
}

function MeterStatusCard({
  meters,
  isLoading,
}: {
  meters: MeterSummary[] | undefined
  isLoading: boolean
}) {
  const count = (status: MeterSummary["status"]) =>
    meters?.filter((m) => m.status === status).length ?? 0
  const dismissed =
    meters?.filter((m) => m.anomaly_type === "FALSE_POSITIVE") ?? []

  return (
    <Card>
      <CardHeader>
        <CardTitle>Estado de los medidores</CardTitle>
        <CardDescription>Según el último análisis.</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {isLoading ? (
          <Skeleton className="h-28 w-full" />
        ) : (
          (["CRITICAL", "ALERT", "NORMAL"] as const).map((status) => (
            <div key={status} className="flex items-center justify-between">
              <MeterStatusBadge status={status} />
              <span className="text-lg font-semibold tabular-nums">
                {count(status)}
              </span>
            </div>
          ))
        )}

        {dismissed.map((meter) => (
          <div
            key={meter.meter_id}
            className="flex gap-2 rounded-lg bg-muted/50 p-3 text-xs text-muted-foreground"
          >
            <ShieldCheckIcon className="mt-0.5 size-4 shrink-0 text-success" />
            <span>
              <span className="font-medium text-foreground">
                {meter.meter_id}
              </span>{" "}
              tuvo una desviación, pero no se escaló: coincide con un evento
              operativo programado.
            </span>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}
