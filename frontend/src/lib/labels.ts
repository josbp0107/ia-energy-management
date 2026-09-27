import type {
  AnalysisStep,
  AnomalyStatus,
  AnomalyType,
  MeterStatus,
  Severity,
} from "@/lib/types"

export const anomalyTypeLabel: Record<AnomalyType, string> = {
  REAL_ANOMALY: "Anomalía real",
  EXPLAINABLE_ANOMALY: "Anomalía explicable",
  FALSE_POSITIVE: "Falso positivo",
  DATA_QUALITY: "Calidad de datos",
}

export const severityLabel: Record<Severity, string> = {
  HIGH: "Alta",
  MEDIUM: "Media",
  LOW: "Baja",
}

export const meterStatusLabel: Record<MeterStatus, string> = {
  CRITICAL: "Crítico",
  ALERT: "Alerta",
  NORMAL: "Normal",
  PENDING: "Sin analizar",
}

export const anomalyStatusLabel: Record<AnomalyStatus, string> = {
  OPEN: "Abierta",
  INVESTIGATING: "En investigación",
  RESOLVED: "Resuelta",
  DISMISSED: "Descartada",
}

export const analysisSteps: {
  step: AnalysisStep
  label: string
  detail: string
}[] = [
  {
    step: "READINGS",
    label: "Lectura de datos",
    detail: "Lecturas horarias y eventos de los 12 medidores",
  },
  {
    step: "BASELINE",
    label: "Baseline",
    detail: "Consumo normal por medidor y hora del día (primera semana)",
  },
  {
    step: "DETECTION",
    label: "Detección",
    detail: "Horas fuera del rango normal y lecturas imposibles",
  },
  {
    step: "CORRELATION",
    label: "Correlación eléctrica",
    detail: "Cambios de corriente y factor de potencia",
  },
  {
    step: "EVENTS",
    label: "Eventos operativos",
    detail: "¿Algún evento a 3 h explica la desviación?",
  },
  {
    step: "EXPLANATION",
    label: "Explicación con IA",
    detail: "Claude redacta el porqué con la evidencia",
  },
  {
    step: "RECOMMENDATION",
    label: "Recomendación",
    detail: "Prioridad y acción sugerida por anomalía",
  },
]
