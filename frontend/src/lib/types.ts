export type Severity = "HIGH" | "MEDIUM" | "LOW"
export type AnomalyType =
  "REAL_ANOMALY" | "EXPLAINABLE_ANOMALY" | "FALSE_POSITIVE" | "DATA_QUALITY"
export type AnomalyStatus = "OPEN" | "INVESTIGATING" | "RESOLVED" | "DISMISSED"
export type MeterStatus = "NORMAL" | "ALERT" | "CRITICAL"
export type RunStatus = "RUNNING" | "COMPLETED" | "FAILED"
export type AnalysisStep =
  | "READINGS"
  | "BASELINE"
  | "DETECTION"
  | "CORRELATION"
  | "EVENTS"
  | "EXPLANATION"
  | "RECOMMENDATION"

export interface User {
  email: string
  name: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface AnalysisRun {
  id: number
  status: RunStatus
  current_step: AnalysisStep
  error?: string
  started_at: string
  finished_at: string | null
  steps?: AnalysisStep[]
}

export interface DashboardSummary {
  total_meters: number
  total_consumption_kwh: number
  period_start: string | null
  period_end: string | null
  anomalies_detected: number
  high_priority: number
  avg_confidence: number
  last_analysis: AnalysisRun | null
}

export interface MeterSummary {
  meter_id: string
  name: string
  location: string
  total_kwh: number
  last_day_kwh: number
  baseline_kwh_day: number | null
  variation_pct: number | null
  readings_count: number
  last_reading_at: string | null
  status: MeterStatus
  anomaly_id: number | null
  anomaly_type: AnomalyType | null
  severity: Severity | null
  priority_score: number | null
  anomaly_status: AnomalyStatus | null
}

export interface Reading {
  id: number
  meter_id: string
  timestamp: string
  consumption_kwh: number
  voltage_v: number
  current_a: number
  power_factor: number
  status: string
}

export interface OperationalEvent {
  id: number
  meter_id: string
  event_timestamp: string
  event_type: string
  description: string
}

export interface HourBaseline {
  hour: number
  kwh: number
  lower_kwh: number
  upper_kwh: number
  current_a: number
  power_factor: number
}

export interface MeterBaseline {
  meter_id: string
  baseline_from: string
  baseline_to: string
  threshold_pct: number
  hours: HourBaseline[]
}

export interface RelatedEvent {
  event_type: string
  description: string
  event_timestamp: string
}

export interface DeviationEvidence {
  start: string
  end: string
  hours_affected: number
  direction: "UP" | "DOWN"
  mean_hourly_deviation_pct: number
  current_change_pct: number
  power_factor_change: number
  baseline_current_a: number
  observed_current_a: number
  baseline_power_factor: number
  observed_power_factor: number
  related_events: RelatedEvent[]
}

export interface DataQualityEvidence {
  invalid_readings: number
  first_invalid: string
  voltage_range: [number, number]
  power_factor_values: number[]
  consumption_stable: boolean
  corroborating_events: string[]
}

export interface ConfidenceFactor {
  label: string
  points: number
  applied: boolean
}

export interface Evidence {
  deviation?: DeviationEvidence
  data_quality?: DataQualityEvidence
  confidence_factors?: ConfidenceFactor[]
}

export interface Anomaly {
  id: number
  analysis_run_id: number
  meter_id: string
  anomaly: boolean
  type: AnomalyType
  severity: Severity
  confidence: number
  reason: string
  recommended_action: string
  explanation_source: "llm" | "template"
  priority_score: number
  variation_pct: number
  baseline_kwh_day: number
  current_kwh_day: number
  evidence: Evidence
  status: AnomalyStatus
  created_at: string
  updated_at: string
}

export interface Rules {
  deviation_threshold_pct: number
  voltage_min: number
  voltage_max: number
  power_factor_drop: number
  current_rise_pct: number
  event_window_hours: number
  min_invalid_readings: number
  baseline_days: number
  severity_weight: Record<string, number>
  explaining_events: string[]
}
