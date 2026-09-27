import { useQuery } from "@tanstack/react-query"
import {
  ArrowRightIcon,
  FlagIcon,
  LightbulbIcon,
  ShieldCheckIcon,
  SparklesIcon,
} from "lucide-react"
import { Link, useParams } from "react-router"

import { ConsumptionChart, MetricChart } from "@/components/meter-charts"
import {
  AnomalyTypeBadge,
  MeterStatusBadge,
  SeverityBadge,
} from "@/components/status-badges"
import { BackLink, ErrorAlert } from "@/components/page"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { api, ApiError } from "@/lib/api"
import {
  formatConfidence,
  formatKWh,
  formatNumber,
  formatPct,
  formatPlantDateTime,
} from "@/lib/format"
import type { Anomaly } from "@/lib/types"
import { useMeterSeries } from "@/lib/use-meter-series"
import { useRules } from "@/lib/use-rules"
import { cn } from "@/lib/utils"

export function MeterDetailPage() {
  const { meterId = "" } = useParams()

  const meter = useQuery({
    queryKey: ["meter", meterId],
    queryFn: () => api.meter(meterId),
  })
  const { rows, ticks, events, error: seriesError } = useMeterSeries(meterId)
  const rules = useRules()
  const threshold = rules?.deviation_threshold_pct
  const anomalyId = meter.data?.anomaly_id
  const anomaly = useQuery({
    queryKey: ["anomaly", anomalyId],
    queryFn: () => api.anomaly(anomalyId!),
    enabled: Boolean(anomalyId),
  })

  if (meter.error instanceof ApiError && meter.error.status === 404) {
    return (
      <ErrorAlert title="Medidor no encontrado">
        No existe el medidor {meterId}.{" "}
        <Link to="/meters" className="underline">
          Volver a medidores
        </Link>
      </ErrorAlert>
    )
  }

  const error = meter.error ?? seriesError
  const m = meter.data
  const deviation = anomaly.data?.evidence.deviation
  const detection =
    deviation && anomaly.data?.anomaly
      ? {
          start: Date.parse(deviation.start),
          end: Date.parse(deviation.end),
          label: `IA · ${deviation.hours_affected} h fuera de lo normal`,
        }
      : undefined

  return (
    <>
      <div className="flex flex-col gap-3">
        <BackLink to="/meters">Medidores</BackLink>
        {m ? (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
            <h1 className="text-2xl font-semibold tracking-tight">
              {m.meter_id}
            </h1>
            <span className="text-muted-foreground">
              {m.name} · {m.location}
            </span>
            <MeterStatusBadge status={m.status} />
            {m.anomaly_type && <AnomalyTypeBadge type={m.anomaly_type} />}
          </div>
        ) : (
          <Skeleton className="h-8 w-80" />
        )}
      </div>

      {error && (
        <ErrorAlert title="No se pudieron cargar los datos" error={error} />
      )}

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Stat
          label="Consumo último día"
          value={m && formatKWh(m.last_day_kwh)}
        />
        <Stat
          label="Baseline diario"
          value={
            m &&
            (m.baseline_kwh_day === null ? "—" : formatKWh(m.baseline_kwh_day))
          }
          hint="mediana de la primera semana"
        />
        <Stat
          label="Variación"
          value={
            m &&
            (m.variation_pct === null
              ? "—"
              : formatPct(m.variation_pct, { signed: true }))
          }
          hint="último día vs baseline"
          alert={
            m && threshold !== undefined
              ? Math.abs(m.variation_pct ?? 0) > threshold
              : false
          }
        />
        <Stat
          label="Prioridad IA"
          value={
            m &&
            (m.priority_score === null
              ? "—"
              : formatNumber(m.priority_score, 2))
          }
          hint={
            anomaly.data
              ? `confianza ${formatConfidence(anomaly.data.confidence)}`
              : "sin anomalía"
          }
        />
      </div>

      {anomaly.data && <FindingCard anomaly={anomaly.data} />}

      <Card>
        <CardHeader>
          <CardTitle>Consumo por hora</CardTitle>
          <CardDescription>
            La banda gris es el rango normal de cada hora del día: su consumo de
            la primera semana, con un margen de {threshold ?? "…"} % hacia
            arriba o hacia abajo. Las horas fuera de la banda son las que el
            motor marca como desviadas.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {rows.length === 0 ? (
            <Skeleton className="h-72 w-full" />
          ) : (
            <ConsumptionChart
              rows={rows}
              ticks={ticks}
              events={events}
              detection={detection}
              thresholdPct={threshold}
            />
          )}
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-3">
        <MetricCard
          title="Voltaje"
          description={
            rules
              ? `Válido entre ${rules.voltage_min} y ${rules.voltage_max} V.`
              : "Rango válido de la red."
          }
        >
          {rows.length > 0 && (
            <MetricChart
              rows={rows}
              ticks={ticks}
              metric="voltage"
              unit="V"
              decimals={1}
              validRange={
                rules ? [rules.voltage_min, rules.voltage_max] : undefined
              }
            />
          )}
        </MetricCard>
        <MetricCard
          title="Corriente"
          description="Frente a su baseline horario."
        >
          {rows.length > 0 && (
            <MetricChart
              rows={rows}
              ticks={ticks}
              metric="current"
              unit="A"
              decimals={1}
              baselineKey="baselineCurrent"
            />
          )}
        </MetricCard>
        <MetricCard
          title="Factor de potencia"
          description="Frente a su baseline horario."
        >
          {rows.length > 0 && (
            <MetricChart
              rows={rows}
              ticks={ticks}
              metric="pf"
              unit=""
              decimals={3}
              baselineKey="baselinePf"
            />
          )}
        </MetricCard>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Eventos operativos</CardTitle>
          <CardDescription>
            Reportados para este medidor (marcados con ⚑ en la gráfica).
          </CardDescription>
        </CardHeader>
        <CardContent>
          {events.length === 0 && (
            <p className="text-sm text-muted-foreground">
              Sin eventos reportados.
            </p>
          )}
          <ul className="flex flex-col gap-2">
            {events.map((event) => (
              <li
                key={event.id}
                className="flex items-start gap-3 rounded-lg border p-3 text-sm"
              >
                <FlagIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
                <div className="flex flex-col gap-0.5">
                  <span className="font-medium">
                    {event.event_type}{" "}
                    <span className="font-normal text-muted-foreground">
                      · {formatPlantDateTime(event.event_timestamp)}
                    </span>
                  </span>
                  <span className="text-muted-foreground">
                    {event.description}
                  </span>
                </div>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>
    </>
  )
}

function Stat({
  label,
  value,
  hint,
  alert = false,
}: {
  label: string
  value: string | undefined
  hint?: string
  alert?: boolean
}) {
  return (
    <Card className="gap-1 py-4">
      <CardContent className="flex flex-col gap-1 px-4">
        <span className="text-xs text-muted-foreground">{label}</span>
        {value === undefined ? (
          <Skeleton className="h-7 w-24" />
        ) : (
          <span
            className={cn("text-xl font-semibold", alert && "text-destructive")}
          >
            {value}
          </span>
        )}
        {hint && <span className="text-xs text-muted-foreground">{hint}</span>}
      </CardContent>
    </Card>
  )
}

function MetricCard({
  title,
  description,
  children,
}: {
  title: string
  description: string
  children: React.ReactNode
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        {children ?? <Skeleton className="h-44 w-full" />}
      </CardContent>
    </Card>
  )
}

function FindingCard({ anomaly }: { anomaly: Anomaly }) {
  const dismissed = !anomaly.anomaly
  return (
    <Card
      className={cn(
        !dismissed && anomaly.severity === "HIGH" && "border-destructive/40"
      )}
    >
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          {dismissed ? (
            <ShieldCheckIcon className="size-5 text-success" />
          ) : (
            <SparklesIcon className="size-5 text-primary" />
          )}
          <CardTitle>
            {dismissed
              ? "La IA descartó esta desviación"
              : "Lo que encontró la IA"}
          </CardTitle>
          <AnomalyTypeBadge type={anomaly.type} />
          {!dismissed && <SeverityBadge severity={anomaly.severity} />}
          <Badge variant="outline" className="ml-auto">
            {anomaly.explanation_source === "llm"
              ? "Explicado por IA (Claude)"
              : "Explicación por plantilla"}
          </Badge>
        </div>
        <CardDescription className="flex flex-wrap items-center justify-between gap-2">
          <span>
            Confianza {formatConfidence(anomaly.confidence)} · prioridad{" "}
            {formatNumber(anomaly.priority_score, 2)}
          </span>
          <Button asChild size="sm" variant="outline">
            <Link to={`/anomalies/${anomaly.id}`}>
              Abrir investigación
              <ArrowRightIcon />
            </Link>
          </Button>
        </CardDescription>
      </CardHeader>
      <CardContent className="grid gap-4 md:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <span className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Por qué
          </span>
          <p className="text-sm leading-relaxed">{anomaly.reason}</p>
        </div>
        <div className="flex flex-col gap-1.5 rounded-lg bg-muted/50 p-3">
          <span className="flex items-center gap-1.5 text-xs font-medium tracking-wide text-muted-foreground uppercase">
            <LightbulbIcon className="size-3.5" />
            Acción recomendada
          </span>
          <p className="text-sm leading-relaxed">
            {anomaly.recommended_action}
          </p>
        </div>
      </CardContent>
    </Card>
  )
}
