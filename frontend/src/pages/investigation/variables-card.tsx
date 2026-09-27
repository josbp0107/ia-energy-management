import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  formatKWh,
  formatNumber,
  formatPct,
  formatPlantDateTime,
} from "@/lib/format"
import type { Anomaly, Rules } from "@/lib/types"
import { useRules } from "@/lib/use-rules"
import { cn } from "@/lib/utils"

interface VariableRow {
  label: string
  baseline: string
  observed: string
  change: string
  alert: boolean
}

function deviationRows(
  anomaly: Anomaly,
  rules: Rules | undefined
): VariableRow[] {
  const dev = anomaly.evidence.deviation!
  const threshold = rules?.deviation_threshold_pct ?? Infinity
  return [
    {
      label: "Consumo diario",
      baseline: formatKWh(anomaly.baseline_kwh_day),
      observed: formatKWh(anomaly.current_kwh_day),
      change: formatPct(anomaly.variation_pct, { signed: true }),
      alert: Math.abs(anomaly.variation_pct) > threshold,
    },
    {
      label: "Consumo horario (desviación media)",
      baseline: rules ? `margen de ${threshold} %` : "—",
      observed: "—",
      change: formatPct(dev.mean_hourly_deviation_pct, { signed: true }),
      alert: Math.abs(dev.mean_hourly_deviation_pct) > threshold,
    },
    {
      label: "Corriente",
      baseline: `${formatNumber(dev.baseline_current_a, 1)} A`,
      observed: `${formatNumber(dev.observed_current_a, 1)} A`,
      change: formatPct(dev.current_change_pct, { signed: true }),
      alert:
        Math.abs(dev.current_change_pct) >
        (rules?.current_rise_pct ?? Infinity),
    },
    {
      label: "Factor de potencia",
      baseline: formatNumber(dev.baseline_power_factor, 3),
      observed: formatNumber(dev.observed_power_factor, 3),
      change: `${dev.power_factor_change > 0 ? "+" : ""}${formatNumber(dev.power_factor_change, 3)}`,
      alert: dev.power_factor_change < -(rules?.power_factor_drop ?? Infinity),
    },
  ]
}

function dataQualityRows(
  anomaly: Anomaly,
  rules: Rules | undefined
): VariableRow[] {
  const dq = anomaly.evidence.data_quality!
  return [
    {
      label: "Voltaje",
      baseline: rules
        ? `válido ${rules.voltage_min}–${rules.voltage_max} V`
        : "—",
      observed: `${formatNumber(dq.voltage_range[0], 1)}–${formatNumber(dq.voltage_range[1], 1)} V`,
      change: "fuera de rango",
      alert: true,
    },
    {
      label: "Factor de potencia",
      baseline: "continuo, ≤ 1",
      observed: dq.power_factor_values
        .map((v) => formatNumber(v, 2))
        .join(" / "),
      change: "valores repetidos",
      alert: true,
    },
    {
      label: "Consumo diario",
      baseline: formatKWh(anomaly.baseline_kwh_day),
      observed: formatKWh(anomaly.current_kwh_day),
      change: `${formatPct(anomaly.variation_pct, { signed: true })} (estable)`,
      alert: false,
    },
  ]
}

export function VariablesCard({ anomaly }: { anomaly: Anomaly }) {
  const rules = useRules()
  const dev = anomaly.evidence.deviation
  const dq = anomaly.evidence.data_quality

  const rows = dev
    ? deviationRows(anomaly, rules)
    : dq
      ? dataQualityRows(anomaly, rules)
      : []
  const window = dev
    ? `${formatPlantDateTime(dev.start)} → ${formatPlantDateTime(dev.end)} · ${dev.hours_affected} h · ${dev.direction === "UP" ? "subida" : "caída"}`
    : dq
      ? `Desde ${formatPlantDateTime(dq.first_invalid)} · ${dq.invalid_readings} lecturas inválidas`
      : null

  return (
    <Card>
      <CardHeader>
        <CardTitle>Variables que cambiaron</CardTitle>
        {window && <CardDescription>{window}</CardDescription>}
      </CardHeader>
      <CardContent className="px-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="pl-6">Variable</TableHead>
              <TableHead className="text-right">Baseline</TableHead>
              <TableHead className="text-right">Observado</TableHead>
              <TableHead className="pr-6 text-right">Cambio</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.label}>
                <TableCell className="pl-6 font-medium">{row.label}</TableCell>
                <TableCell className="text-right text-muted-foreground tabular-nums">
                  {row.baseline}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {row.observed}
                </TableCell>
                <TableCell
                  className={cn(
                    "pr-6 text-right tabular-nums",
                    row.alert
                      ? "font-semibold text-destructive"
                      : "text-muted-foreground"
                  )}
                >
                  {row.change}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}
