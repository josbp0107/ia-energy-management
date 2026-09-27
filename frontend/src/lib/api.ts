import { clearSession, loadSession } from "@/lib/session"
import type {
  AnalysisRun,
  Anomaly,
  AnomalyStatus,
  DashboardSummary,
  LoginResponse,
  MeterBaseline,
  MeterSummary,
  OperationalEvent,
  Reading,
  Rules,
} from "@/lib/types"

const API_URL = import.meta.env.VITE_API_URL as string | undefined
if (!API_URL) {
  throw new Error(
    "Falta VITE_API_URL: copia frontend/.env.example a frontend/.env.local"
  )
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = "ApiError"
    this.status = status
  }
}

let unauthorizedHandler: (() => void) | null = null
export function setUnauthorizedHandler(handler: (() => void) | null) {
  unauthorizedHandler = handler
}

interface RequestOptions {
  method?: "GET" | "POST" | "PATCH"
  body?: unknown
  auth?: boolean
}

async function request<T>(
  path: string,
  { method = "GET", body, auth = true }: RequestOptions = {}
): Promise<T> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers["Content-Type"] = "application/json"
  if (auth) {
    const token = loadSession()?.token
    if (token) headers.Authorization = `Bearer ${token}`
  }

  let response: Response
  try {
    response = await fetch(`${API_URL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(
      0,
      "No se pudo conectar con el servidor. ¿Está corriendo el backend?"
    )
  }

  if (!response.ok) {
    const data = (await response.json().catch(() => null)) as {
      error?: string
    } | null
    if (response.status === 401 && auth) {
      clearSession()
      unauthorizedHandler?.()
    }
    throw new ApiError(
      response.status,
      data?.error ?? `Error ${response.status}`
    )
  }

  return (await response.json()) as T
}

function query(params: Record<string, string | undefined>) {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value) search.set(key, value)
  }
  const text = search.toString()
  return text ? `?${text}` : ""
}

const id = (value: string | number) => encodeURIComponent(String(value))

export const api = {
  login: (email: string, password: string) =>
    request<LoginResponse>("/auth/login", {
      method: "POST",
      body: { email, password },
      auth: false,
    }),

  dashboardSummary: () => request<DashboardSummary>("/dashboard/summary"),

  meters: () => request<MeterSummary[]>("/meters"),
  meter: (meterId: string) => request<MeterSummary>(`/meters/${id(meterId)}`),
  meterReadings: (meterId: string) =>
    request<Reading[]>(`/meters/${id(meterId)}/readings`),
  meterBaseline: (meterId: string) =>
    request<MeterBaseline>(`/meters/${id(meterId)}/baseline`),

  events: (meterId?: string) =>
    request<OperationalEvent[]>(`/events${query({ meter_id: meterId })}`),

  anomalies: (
    filters: {
      meter_id?: string
      status?: AnomalyStatus
      severity?: string
    } = {}
  ) => request<Anomaly[]>(`/anomalies${query(filters)}`),
  anomaly: (anomalyId: number) =>
    request<Anomaly>(`/anomalies/${id(anomalyId)}`),
  updateAnomalyStatus: (anomalyId: number, status: AnomalyStatus) =>
    request<Anomaly>(`/anomalies/${id(anomalyId)}`, {
      method: "PATCH",
      body: { status },
    }),

  rules: () => request<Rules>("/analysis/rules"),
  startAnalysis: () => request<AnalysisRun>("/ai/analyze", { method: "POST" }),
  analysis: (runId: number) =>
    request<AnalysisRun>(`/ai/analysis/${id(runId)}`),
}
