import { CheckCircle2Icon, CircleIcon } from "lucide-react"

import { SeverityBadge } from "@/components/status-badges"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { formatConfidence, formatNumber } from "@/lib/format"
import { severityLabel } from "@/lib/labels"
import type { Anomaly } from "@/lib/types"
import { useRules } from "@/lib/use-rules"
import { cn } from "@/lib/utils"

export function ScoreCard({ anomaly }: { anomaly: Anomaly }) {
  const rules = useRules()
  const factors = anomaly.evidence.confidence_factors
  const weight = rules?.severity_weight[anomaly.severity]

  return (
    <Card>
      <CardHeader>
        <CardTitle>Severidad, confianza y prioridad</CardTitle>
        <CardDescription>
          Calculadas por el motor estadístico, no por el LLM.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-5">
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Severidad</span>
          <span className="flex items-center gap-2">
            <SeverityBadge severity={anomaly.severity} />
            {weight !== undefined && (
              <span className="text-xs text-muted-foreground">
                peso {weight}
              </span>
            )}
          </span>
        </div>

        <div className="flex flex-col gap-2">
          <div className="flex items-baseline justify-between">
            <span className="text-sm text-muted-foreground">Confianza</span>
            <span className="text-2xl font-semibold">
              {formatConfidence(anomaly.confidence)}
            </span>
          </div>
          {factors ? (
            <ul className="flex flex-col gap-1.5">
              {factors.map((f) => (
                <li
                  key={f.label}
                  className={cn(
                    "flex items-start gap-2 text-sm",
                    !f.applied && "text-muted-foreground"
                  )}
                >
                  {f.applied ? (
                    <CheckCircle2Icon className="mt-0.5 size-4 shrink-0 text-success" />
                  ) : (
                    <CircleIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground/40" />
                  )}
                  <span className="flex-1">{f.label}</span>
                  <span className="tabular-nums">
                    +{formatNumber(f.points, 2)}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-xs text-muted-foreground">
              Ejecuta un nuevo análisis para ver el desglose de la confianza.
            </p>
          )}
        </div>

        <div className="flex flex-col gap-1.5 rounded-lg bg-muted/50 p-3">
          <div className="flex items-baseline justify-between">
            <span className="text-sm text-muted-foreground">Prioridad</span>
            <span className="text-2xl font-semibold">
              {formatNumber(anomaly.priority_score, 2)}
            </span>
          </div>
          {weight !== undefined && (
            <p className="text-xs text-muted-foreground">
              {weight} ({severityLabel[anomaly.severity].toLowerCase()}) ×{" "}
              {formatNumber(anomaly.confidence, 2)} × (1 +{" "}
              {formatNumber(Math.abs(anomaly.variation_pct), 1)} %) ={" "}
              <span className="font-medium text-foreground">
                {formatNumber(anomaly.priority_score, 2)}
              </span>
            </p>
          )}
          <p className="text-xs text-muted-foreground">
            severidad × confianza × magnitud del cambio
          </p>
        </div>
      </CardContent>
    </Card>
  )
}
