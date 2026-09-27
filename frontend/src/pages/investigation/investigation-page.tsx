import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { ArrowRightIcon, SearchIcon } from "lucide-react"
import { Link, useParams } from "react-router"
import { toast } from "sonner"

import { ConsumptionChart } from "@/components/meter-charts"
import { BackLink, ErrorAlert } from "@/components/page"
import {
  AnomalyStatusBadge,
  AnomalyTypeBadge,
  SeverityBadge,
} from "@/components/status-badges"
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
import { anomalyStatusLabel } from "@/lib/labels"
import type { AnomalyStatus } from "@/lib/types"
import { useMeterSeries } from "@/lib/use-meter-series"
import { useRules } from "@/lib/use-rules"

import { ActionCard } from "./action-card"
import { EventsCard } from "./events-card"
import { EvidenceCard } from "./evidence-card"
import { FindingCard } from "./finding-card"
import { ScoreCard } from "./score-card"
import { VariablesCard } from "./variables-card"

export function InvestigationPage() {
  const anomalyId = Number(useParams().id)
  const queryClient = useQueryClient()
  const rules = useRules()

  const anomaly = useQuery({
    queryKey: ["anomaly", anomalyId],
    queryFn: () => api.anomaly(anomalyId),
    enabled: Number.isInteger(anomalyId) && anomalyId > 0,
  })
  const a = anomaly.data
  const meter = useQuery({
    queryKey: ["meter", a?.meter_id],
    queryFn: () => api.meter(a!.meter_id),
    enabled: Boolean(a),
  })
  const series = useMeterSeries(a?.meter_id)

  const updateStatus = useMutation({
    mutationFn: (status: AnomalyStatus) =>
      api.updateAnomalyStatus(anomalyId, status),
    onSuccess: (updated) => {
      queryClient.setQueryData(["anomaly", anomalyId], updated)
      queryClient.invalidateQueries({ queryKey: ["anomalies"] })
      queryClient.invalidateQueries({ queryKey: ["meters"] })
      queryClient.invalidateQueries({ queryKey: ["meter", updated.meter_id] })
      toast.success(
        `Anomalía ${updated.meter_id}: ${anomalyStatusLabel[updated.status].toLowerCase()}`
      )
    },
    onError: (error) =>
      toast.error("No se pudo actualizar el estado", {
        description: error.message,
      }),
  })

  if (
    !Number.isInteger(anomalyId) ||
    (anomaly.error instanceof ApiError && anomaly.error.status === 404)
  ) {
    return (
      <ErrorAlert title="Anomalía no encontrada">
        <Link to="/anomalies" className="underline">
          Volver a Anomalías IA
        </Link>
      </ErrorAlert>
    )
  }

  if (!a) {
    return (
      <div className="flex flex-col gap-4">
        {anomaly.error ? (
          <ErrorAlert
            title="No se pudo cargar la anomalía"
            error={anomaly.error}
          />
        ) : (
          <>
            <Skeleton className="h-10 w-96" />
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-72 w-full" />
          </>
        )}
      </div>
    )
  }

  const deviation = a.evidence.deviation
  const dismissedByEngine = !a.anomaly

  return (
    <>
      <div className="flex flex-col gap-3">
        <BackLink to="/anomalies">Anomalías IA</BackLink>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
          <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <SearchIcon className="size-4" />
            Investigación
          </span>
          <h1 className="text-2xl font-semibold tracking-tight">
            {a.meter_id}
          </h1>
          {meter.data && (
            <span className="text-muted-foreground">
              {meter.data.name} · {meter.data.location}
            </span>
          )}
          <AnomalyTypeBadge type={a.type} />
          {!dismissedByEngine && <SeverityBadge severity={a.severity} />}
          <AnomalyStatusBadge status={a.status} />
          <Button asChild variant="outline" size="sm" className="ml-auto">
            <Link to={`/meters/${a.meter_id}`}>
              Ver medidor
              <ArrowRightIcon />
            </Link>
          </Button>
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        <div className="flex flex-col gap-4 xl:col-span-2">
          <FindingCard anomaly={a} />
          <ActionCard
            anomaly={a}
            onChange={(status) => updateStatus.mutate(status)}
            isPending={updateStatus.isPending}
          />
          <VariablesCard anomaly={a} />

          <Card>
            <CardHeader>
              <CardTitle>Consumo frente al baseline</CardTitle>
              <CardDescription>
                Banda gris: rango normal de cada hora, con un margen de{" "}
                {rules?.deviation_threshold_pct ?? "…"} % sobre o bajo el
                baseline. Sombreado: ventana que detectó la IA.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {series.rows.length === 0 ? (
                <Skeleton className="h-72 w-full" />
              ) : (
                <ConsumptionChart
                  rows={series.rows}
                  ticks={series.ticks}
                  events={series.events}
                  thresholdPct={rules?.deviation_threshold_pct}
                  detection={
                    deviation && !dismissedByEngine
                      ? {
                          start: Date.parse(deviation.start),
                          end: Date.parse(deviation.end),
                          label: `IA · ${deviation.hours_affected} h fuera de lo normal`,
                        }
                      : undefined
                  }
                />
              )}
            </CardContent>
          </Card>

          <EventsCard anomaly={a} events={series.events} />
        </div>

        <div className="flex flex-col gap-4">
          <ScoreCard anomaly={a} />
          <EvidenceCard anomaly={a} />
        </div>
      </div>
    </>
  )
}
