const LOCALE = "es-CO"

export function formatNumber(value: number, decimals = 1) {
  return new Intl.NumberFormat(LOCALE, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(value)
}

export function formatKWh(value: number, decimals = 1) {
  return `${formatNumber(value, decimals)} kWh`
}

export function formatPct(
  value: number,
  { signed = false, decimals = 1 } = {}
) {
  const sign = signed && value > 0 ? "+" : ""
  return `${sign}${formatNumber(value, decimals)} %`
}

export function formatConfidence(value: number) {
  return `${formatNumber(value * 100, 0)} %`
}

const plantDateTime = new Intl.DateTimeFormat(LOCALE, {
  timeZone: "UTC",
  day: "2-digit",
  month: "short",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
})
const plantDate = new Intl.DateTimeFormat(LOCALE, {
  timeZone: "UTC",
  day: "2-digit",
  month: "short",
  year: "numeric",
})

export const formatPlantDateTime = (iso: string) =>
  plantDateTime.format(new Date(iso))
export const formatPlantDate = (iso: string) => plantDate.format(new Date(iso))

const localDateTime = new Intl.DateTimeFormat(LOCALE, {
  day: "2-digit",
  month: "short",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
})
export const formatLocalDateTime = (iso: string) =>
  localDateTime.format(new Date(iso))
