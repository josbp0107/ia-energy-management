import { LightbulbIcon } from "lucide-react"

import { AnomalyStatusBadge } from "@/components/status-badges"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { Anomaly, AnomalyStatus } from "@/lib/types"

interface StatusOption {
  status: AnomalyStatus
  label: string
  primary?: boolean
}

function nextOptions(status: AnomalyStatus): StatusOption[] {
  if (status === "OPEN") {
    return [
      { status: "INVESTIGATING", label: "Investigar", primary: true },
      { status: "RESOLVED", label: "Marcar resuelta" },
      { status: "DISMISSED", label: "Descartar" },
    ]
  }
  if (status === "INVESTIGATING") {
    return [
      { status: "RESOLVED", label: "Marcar resuelta", primary: true },
      { status: "DISMISSED", label: "Descartar" },
      { status: "OPEN", label: "Reabrir" },
    ]
  }
  return [{ status: "OPEN", label: "Reabrir" }]
}

export function ActionCard({
  anomaly,
  onChange,
  isPending,
}: {
  anomaly: Anomaly
  onChange: (status: AnomalyStatus) => void
  isPending: boolean
}) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <LightbulbIcon className="size-5 text-warning" />
          <CardTitle>Acción recomendada</CardTitle>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <p className="leading-relaxed">{anomaly.recommended_action}</p>
        <div className="flex flex-wrap items-center gap-2 border-t pt-4">
          <span className="mr-1 text-sm text-muted-foreground">
            Estado: <AnomalyStatusBadge status={anomaly.status} />
          </span>
          {nextOptions(anomaly.status).map((option) => (
            <Button
              key={option.status}
              size="sm"
              variant={option.primary ? "default" : "outline"}
              disabled={isPending}
              onClick={() => onChange(option.status)}
            >
              {option.label}
            </Button>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
