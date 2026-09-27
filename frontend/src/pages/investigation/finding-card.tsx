import { SparklesIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { Anomaly } from "@/lib/types"
import { cn } from "@/lib/utils"

export function FindingCard({ anomaly }: { anomaly: Anomaly }) {
  return (
    <Card
      className={cn(
        anomaly.anomaly &&
          anomaly.severity === "HIGH" &&
          "border-destructive/40"
      )}
    >
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <SparklesIcon className="size-5 text-primary" />
          <CardTitle>Qué encontró la IA</CardTitle>
          <Badge variant="outline" className="ml-auto">
            {anomaly.explanation_source === "llm"
              ? "Explicado por IA (Claude)"
              : "Explicación por plantilla"}
          </Badge>
        </div>
      </CardHeader>
      <CardContent>
        <p className="leading-relaxed">{anomaly.reason}</p>
      </CardContent>
    </Card>
  )
}
