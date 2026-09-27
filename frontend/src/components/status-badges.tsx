import type { ComponentProps } from "react"

import { Badge } from "@/components/ui/badge"
import {
  anomalyStatusLabel,
  anomalyTypeLabel,
  meterStatusLabel,
  severityLabel,
} from "@/lib/labels"
import type {
  AnomalyStatus,
  AnomalyType,
  MeterStatus,
  Severity,
} from "@/lib/types"

type Variant = ComponentProps<typeof Badge>["variant"]

const severityVariant: Record<Severity, Variant> = {
  HIGH: "destructive",
  MEDIUM: "warning",
  LOW: "secondary",
}

const typeVariant: Record<AnomalyType, Variant> = {
  REAL_ANOMALY: "destructive",
  DATA_QUALITY: "data-quality",
  EXPLAINABLE_ANOMALY: "warning",
  FALSE_POSITIVE: "secondary",
}

const meterStatusVariant: Record<MeterStatus, Variant> = {
  CRITICAL: "destructive",
  ALERT: "warning",
  NORMAL: "success",
  PENDING: "outline",
}

const anomalyStatusVariant: Record<AnomalyStatus, Variant> = {
  OPEN: "outline",
  INVESTIGATING: "info",
  RESOLVED: "success",
  DISMISSED: "secondary",
}

export function SeverityBadge({ severity }: { severity: Severity }) {
  return (
    <Badge variant={severityVariant[severity]}>{severityLabel[severity]}</Badge>
  )
}

export function AnomalyTypeBadge({ type }: { type: AnomalyType }) {
  return <Badge variant={typeVariant[type]}>{anomalyTypeLabel[type]}</Badge>
}

export function MeterStatusBadge({ status }: { status: MeterStatus }) {
  return (
    <Badge variant={meterStatusVariant[status]}>
      {meterStatusLabel[status]}
    </Badge>
  )
}

export function AnomalyStatusBadge({ status }: { status: AnomalyStatus }) {
  return (
    <Badge variant={anomalyStatusVariant[status]}>
      {anomalyStatusLabel[status]}
    </Badge>
  )
}
