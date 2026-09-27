import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import type { Anomaly } from "@/lib/types"

export function EvidenceCard({ anomaly }: { anomaly: Anomaly }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Evidencia completa</CardTitle>
        <CardDescription>
          Datos exactos del motor, el LLM solo puede citar cifras de aquí.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <details>
          <summary className="cursor-pointer text-sm font-medium select-none">
            Ver JSON
          </summary>
          <pre className="mt-3 max-h-96 overflow-auto rounded-lg bg-muted p-3 text-xs">
            {JSON.stringify(anomaly.evidence, null, 2)}
          </pre>
        </details>
      </CardContent>
    </Card>
  )
}
