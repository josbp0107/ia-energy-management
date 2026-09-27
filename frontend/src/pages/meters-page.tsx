import { useQuery } from "@tanstack/react-query"
import { ChevronRightIcon, SearchIcon, ShieldCheckIcon } from "lucide-react"
import { useMemo, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router"

import { AnomalyTypeBadge, MeterStatusBadge } from "@/components/status-badges"
import { ErrorAlert, PageHeader } from "@/components/page"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { api } from "@/lib/api"
import { formatKWh, formatPct } from "@/lib/format"
import type { MeterStatus, MeterSummary } from "@/lib/types"
import { useRules } from "@/lib/use-rules"
import { cn } from "@/lib/utils"

const FILTERS = {
  all: { label: "Todos", match: () => true },
  normal: {
    label: "Normales",
    match: (m: MeterSummary) => m.status === "NORMAL",
  },
  alerts: {
    label: "Alertas",
    match: (m: MeterSummary) => m.status === "ALERT" || m.status === "CRITICAL",
  },
  critical: {
    label: "Críticas",
    match: (m: MeterSummary) => m.status === "CRITICAL",
  },
} as const
type FilterKey = keyof typeof FILTERS

const statusRank: Record<MeterStatus, number> = {
  CRITICAL: 2,
  ALERT: 1,
  NORMAL: 0,
  PENDING: 0,
}

const SORTS = {
  severity: {
    label: "Severidad",
    compare: (a: MeterSummary, b: MeterSummary) =>
      statusRank[b.status] - statusRank[a.status] ||
      (b.priority_score ?? 0) - (a.priority_score ?? 0),
  },
  variation: {
    label: "Variación",
    compare: (a: MeterSummary, b: MeterSummary) =>
      Math.abs(b.variation_pct ?? 0) - Math.abs(a.variation_pct ?? 0),
  },
  consumption: {
    label: "Consumo",
    compare: (a: MeterSummary, b: MeterSummary) =>
      b.last_day_kwh - a.last_day_kwh,
  },
} as const
type SortKey = keyof typeof SORTS

const isFilter = (v: string | null): v is FilterKey =>
  v !== null && v in FILTERS
const isSort = (v: string | null): v is SortKey => v !== null && v in SORTS

function matchesSearch(meter: MeterSummary, search: string) {
  const text = search.trim().toLowerCase()
  if (!text) return true
  return [meter.meter_id, meter.name, meter.location].some((field) =>
    field.toLowerCase().includes(text)
  )
}

export function MetersPage() {
  const navigate = useNavigate()
  const meters = useQuery({ queryKey: ["meters"], queryFn: api.meters })
  const rules = useRules()

  const [params, setParams] = useSearchParams()
  const filter: FilterKey = isFilter(params.get("filter"))
    ? (params.get("filter") as FilterKey)
    : "all"
  const sort: SortKey = isSort(params.get("sort"))
    ? (params.get("sort") as SortKey)
    : "severity"
  const [search, setSearch] = useState(() => params.get("q") ?? "")

  function updateParam(key: string, value: string, defaultValue: string) {
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        if (value === defaultValue) next.delete(key)
        else next.set(key, value)
        return next
      },
      { replace: true }
    )
  }

  const all = useMemo(() => meters.data ?? [], [meters.data])
  const visible = useMemo(
    () =>
      all
        .filter((m) => FILTERS[filter].match(m) && matchesSearch(m, search))
        .sort(SORTS[sort].compare),
    [all, filter, search, sort]
  )

  return (
    <>
      <PageHeader
        title="Medidores"
        description="Consumo del último día frente a su baseline y estado según el último análisis de IA."
      />

      {meters.error && (
        <ErrorAlert
          title="No se pudieron cargar los medidores"
          error={meters.error}
        />
      )}

      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <ToggleGroup
          type="single"
          variant="outline"
          value={filter}
          onValueChange={(value) =>
            value && updateParam("filter", value, "all")
          }
        >
          {(Object.keys(FILTERS) as FilterKey[]).map((key) => (
            <ToggleGroupItem key={key} value={key} className="gap-1.5 px-3">
              {FILTERS[key].label}
              <span className="text-xs text-muted-foreground tabular-nums">
                {all.filter(FILTERS[key].match).length}
              </span>
            </ToggleGroupItem>
          ))}
        </ToggleGroup>

        <div className="flex gap-2">
          <div className="relative flex-1 lg:w-64 lg:flex-none">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Buscar medidor (ej. M-109)"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value)
                updateParam("q", e.target.value, "")
              }}
              className="pl-8"
              aria-label="Buscar medidor"
            />
          </div>
          <Select
            value={sort}
            onValueChange={(value) => updateParam("sort", value, "severity")}
          >
            <SelectTrigger className="w-44" aria-label="Ordenar por">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(Object.keys(SORTS) as SortKey[]).map((key) => (
                <SelectItem key={key} value={key}>
                  Ordenar: {SORTS[key].label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <Card className="py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="pl-4">Medidor</TableHead>
                <TableHead className="hidden md:table-cell">
                  Ubicación
                </TableHead>
                <TableHead className="text-right">Último día</TableHead>
                <TableHead className="hidden text-right sm:table-cell">
                  Baseline/día
                </TableHead>
                <TableHead className="text-right">Variación</TableHead>
                <TableHead>Estado</TableHead>
                <TableHead className="hidden lg:table-cell">
                  Diagnóstico IA
                </TableHead>
                <TableHead className="w-8" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {meters.isPending &&
                Array.from({ length: 6 }, (_, i) => (
                  <TableRow key={i}>
                    <TableCell colSpan={8} className="px-4">
                      <Skeleton className="h-8 w-full" />
                    </TableCell>
                  </TableRow>
                ))}

              {meters.isSuccess && visible.length === 0 && (
                <TableRow>
                  <TableCell
                    colSpan={8}
                    className="py-10 text-center text-muted-foreground"
                  >
                    Ningún medidor coincide con los filtros.
                  </TableCell>
                </TableRow>
              )}

              {visible.map((meter) => (
                <TableRow
                  key={meter.meter_id}
                  onClick={() => navigate(`/meters/${meter.meter_id}`)}
                  className="cursor-pointer"
                >
                  <TableCell className="pl-4">
                    {}
                    <Link
                      to={`/meters/${meter.meter_id}`}
                      className="font-medium hover:underline"
                      onClick={(e) => e.stopPropagation()}
                    >
                      {meter.meter_id}
                    </Link>
                    <div className="text-xs text-muted-foreground">
                      {meter.name}
                    </div>
                  </TableCell>
                  <TableCell className="hidden text-muted-foreground md:table-cell">
                    {meter.location}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {formatKWh(meter.last_day_kwh)}
                  </TableCell>
                  <TableCell className="hidden text-right text-muted-foreground tabular-nums sm:table-cell">
                    {meter.baseline_kwh_day === null
                      ? "—"
                      : formatKWh(meter.baseline_kwh_day)}
                  </TableCell>
                  <TableCell className="text-right">
                    <VariationValue
                      value={meter.variation_pct}
                      threshold={rules?.deviation_threshold_pct}
                    />
                  </TableCell>
                  <TableCell>
                    <MeterStatusBadge status={meter.status} />
                  </TableCell>
                  <TableCell className="hidden lg:table-cell">
                    <Diagnosis meter={meter} />
                  </TableCell>
                  <TableCell>
                    <ChevronRightIcon className="size-4 text-muted-foreground" />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </>
  )
}

function VariationValue({
  value,
  threshold,
}: {
  value: number | null
  threshold: number | undefined
}) {
  if (value === null) return <span className="text-muted-foreground">—</span>
  const significant = threshold !== undefined && Math.abs(value) > threshold
  return (
    <span
      className={cn(
        "tabular-nums",
        significant ? "font-semibold text-destructive" : "text-muted-foreground"
      )}
    >
      {formatPct(value, { signed: true })}
    </span>
  )
}

function Diagnosis({ meter }: { meter: MeterSummary }) {
  if (!meter.anomaly_type) {
    return <span className="text-xs text-muted-foreground">Sin anomalías</span>
  }
  if (meter.anomaly_type === "FALSE_POSITIVE") {
    return (
      <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <ShieldCheckIcon className="size-3.5 text-success" />
        Descartado por evento
      </span>
    )
  }
  return <AnomalyTypeBadge type={meter.anomaly_type} />
}
