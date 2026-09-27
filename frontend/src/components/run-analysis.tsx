import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  AlertTriangleIcon,
  ArrowRightIcon,
  CheckCircle2Icon,
  CircleIcon,
  Loader2Icon,
  SparklesIcon,
} from "lucide-react"
import { useEffect, useRef, useState } from "react"
import { useNavigate } from "react-router"
import { toast } from "sonner"

import { AnomalyTypeBadge } from "@/components/status-badges"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Progress } from "@/components/ui/progress"
import { api } from "@/lib/api"
import { formatNumber } from "@/lib/format"
import { analysisSteps } from "@/lib/labels"
import type { AnalysisRun, Anomaly } from "@/lib/types"
import { cn } from "@/lib/utils"

const POLL_MS = 500

function summarize(anomalies: Anomaly[]) {
  const high = anomalies.filter(
    (a) => a.anomaly && a.severity === "HIGH"
  ).length
  return `${anomalies.length} anomalías detectadas, ${high} de alta prioridad`
}

export function RunAnalysis() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [runId, setRunId] = useState<number | null>(null)
  const [open, setOpen] = useState(false)

  const start = useMutation({
    mutationFn: api.startAnalysis,
    onSuccess: (run) => {
      setRunId(run.id)
      setOpen(true)
    },
    onError: (error) =>
      toast.error("No se pudo iniciar el análisis", {
        description: error.message,
      }),
  })

  const { data: run } = useQuery({
    queryKey: ["analysis", runId],
    queryFn: () => api.analysis(runId!),
    enabled: runId !== null,
    refetchInterval: (query) =>
      query.state.data?.status === "RUNNING" || !query.state.data
        ? POLL_MS
        : false,
  })

  const isRunning = start.isPending || run?.status === "RUNNING"
  const completed = run?.status === "COMPLETED"

  const anomalies = useQuery({
    queryKey: ["anomalies"],
    queryFn: () => api.anomalies(),
    enabled: completed,
  })

  const handledRun = useRef<number | null>(null)
  useEffect(() => {
    if (!run || run.status === "RUNNING" || handledRun.current === run.id) {
      return
    }
    handledRun.current = run.id
    if (run.status === "FAILED") {
      toast.error("El análisis falló", { description: run.error })
      return
    }
    queryClient
      .invalidateQueries({ predicate: (q) => q.queryKey[0] !== "analysis" })
      .then(() =>
        queryClient.fetchQuery({
          queryKey: ["anomalies"],
          queryFn: () => api.anomalies(),
        })
      )
      .then((result) =>
        toast.success("Análisis completado", { description: summarize(result) })
      )
      .catch(() => undefined)
  }, [run, queryClient])

  const currentIndex = run
    ? completed
      ? analysisSteps.length
      : analysisSteps.findIndex((s) => s.step === run.current_step)
    : 0
  const progress = (Math.max(currentIndex, 0) / analysisSteps.length) * 100
  const top = anomalies.data?.[0]

  return (
    <>
      <Button
        size="sm"
        onClick={() => (isRunning ? setOpen(true) : start.mutate())}
        className="ml-auto"
      >
        {isRunning ? (
          <Loader2Icon className="animate-spin" />
        ) : (
          <SparklesIcon />
        )}
        {isRunning
          ? `Analizando… ${Math.min(currentIndex + 1, analysisSteps.length)}/${analysisSteps.length}`
          : "Run AI Analysis"}
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <SparklesIcon className="size-5 text-primary" />
              Análisis de IA
            </DialogTitle>
            <DialogDescription>
              El motor estadístico detecta y prioriza; Claude explica cada
              hallazgo con la evidencia calculada.
            </DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-2">
            <Progress value={progress} />
            <span className="text-right text-xs text-muted-foreground">
              {completed
                ? "Completado"
                : `Paso ${Math.min(currentIndex + 1, analysisSteps.length)} de ${analysisSteps.length}`}
            </span>
          </div>

          <StepList run={run} currentIndex={currentIndex} />

          {run?.status === "FAILED" && (
            <Alert variant="destructive">
              <AlertTriangleIcon />
              <AlertTitle>El análisis falló</AlertTitle>
              <AlertDescription>{run.error}</AlertDescription>
            </Alert>
          )}

          {completed && top && !anomalies.isFetching && (
            <div className="flex flex-col gap-3 rounded-lg border bg-muted/40 p-3">
              <p className="text-sm font-medium">
                {summarize(anomalies.data!)}
              </p>
              <div className="flex items-center gap-2 text-sm">
                <span className="text-muted-foreground">Prioridad 1:</span>
                <span className="font-medium">{top.meter_id}</span>
                <AnomalyTypeBadge type={top.type} />
                <span className="ml-auto text-muted-foreground">
                  {formatNumber(top.priority_score, 2)}
                </span>
              </div>
            </div>
          )}

          <DialogFooter>
            {run?.status === "FAILED" && (
              <Button variant="outline" onClick={() => start.mutate()}>
                Reintentar
              </Button>
            )}
            {completed && top ? (
              <Button
                onClick={() => {
                  setOpen(false)
                  navigate(`/anomalies/${top.id}`)
                }}
              >
                Ver anomalía principal
                <ArrowRightIcon />
              </Button>
            ) : (
              <Button variant="outline" onClick={() => setOpen(false)}>
                {isRunning ? "Seguir en segundo plano" : "Cerrar"}
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

function StepList({
  run,
  currentIndex,
}: {
  run: AnalysisRun | undefined
  currentIndex: number
}) {
  const failed = run?.status === "FAILED"
  return (
    <ol className="flex flex-col gap-2.5">
      {analysisSteps.map((step, index) => {
        const done = index < currentIndex
        const active = index === currentIndex && !failed
        return (
          <li key={step.step} className="flex items-start gap-3">
            {done ? (
              <CheckCircle2Icon className="mt-0.5 size-4 shrink-0 text-success" />
            ) : active ? (
              <Loader2Icon className="mt-0.5 size-4 shrink-0 animate-spin text-primary" />
            ) : failed && index === currentIndex ? (
              <AlertTriangleIcon className="mt-0.5 size-4 shrink-0 text-destructive" />
            ) : (
              <CircleIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground/40" />
            )}
            <div className="flex flex-col">
              <span
                className={cn(
                  "text-sm",
                  !done && !active && "text-muted-foreground"
                )}
              >
                {step.label}
              </span>
              <span className="text-xs text-muted-foreground">
                {step.detail}
              </span>
            </div>
          </li>
        )
      })}
    </ol>
  )
}
