import { FlagIcon } from "lucide-react"
import type { ComponentProps } from "react"

import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { formatPlantDateTime } from "@/lib/format"
import type { Anomaly, OperationalEvent, Rules } from "@/lib/types"
import { useRules } from "@/lib/use-rules"

interface EventRole {
  text: string
  variant: ComponentProps<typeof Badge>["variant"]
}

function eventRole(
  event: OperationalEvent,
  anomaly: Anomaly,
  rules: Rules | undefined
): EventRole {
  if (anomaly.evidence.data_quality && event.event_type === "DATA_QUALITY") {
    return { text: "Corrobora el diagnóstico", variant: "data-quality" }
  }
  const related = anomaly.evidence.deviation?.related_events ?? []
  const inWindow = related.some(
    (r) =>
      r.event_type === event.event_type &&
      Date.parse(r.event_timestamp) === Date.parse(event.event_timestamp)
  )
  if (!inWindow) {
    return {
      text: `Fuera de la ventana de ${rules?.event_window_hours ?? "…"} h`,
      variant: "secondary",
    }
  }
  if (rules?.explaining_events.includes(event.event_type)) {
    return { text: "Explica la desviación", variant: "success" }
  }
  return { text: "No explica la desviación", variant: "destructive" }
}

export function EventsCard({
  anomaly,
  events,
}: {
  anomaly: Anomaly
  events: OperationalEvent[]
}) {
  const rules = useRules()

  return (
    <Card>
      <CardHeader>
        <CardTitle>Eventos operativos</CardTitle>
        <CardDescription>
          Solo {rules?.explaining_events.join(" o ") ?? "ciertos eventos"} a ±
          {rules?.event_window_hours ?? "…"} h del inicio explican una
          desviación. Un evento UNKNOWN no explica nada.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        {events.length === 0 && (
          <p className="text-sm text-muted-foreground">
            Sin eventos reportados para este medidor: nada explica la
            desviación.
          </p>
        )}
        {events.map((event) => {
          const role = eventRole(event, anomaly, rules)
          return (
            <div
              key={event.id}
              className="flex flex-wrap items-start gap-3 rounded-lg border p-3 text-sm"
            >
              <FlagIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
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
              <Badge variant={role.variant}>{role.text}</Badge>
            </div>
          )
        })}
      </CardContent>
    </Card>
  )
}
