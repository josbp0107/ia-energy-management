import { AlertTriangleIcon, ArrowLeftIcon, Loader2Icon } from "lucide-react"
import type { ReactNode } from "react"
import { Link } from "react-router"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"

export function PageHeader({
  title,
  description,
}: {
  title: string
  description?: ReactNode
}) {
  return (
    <div className="flex flex-col gap-1">
      <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
      {description && (
        <p className="text-sm text-muted-foreground">{description}</p>
      )}
    </div>
  )
}

export function BackLink({
  to,
  children,
}: {
  to: string
  children: ReactNode
}) {
  return (
    <Link
      to={to}
      className="flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
    >
      <ArrowLeftIcon className="size-4" />
      {children}
    </Link>
  )
}

export function ErrorAlert({
  title,
  error,
  children,
}: {
  title: string
  error?: Error | null
  children?: ReactNode
}) {
  return (
    <Alert variant="destructive">
      <AlertTriangleIcon />
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>{children ?? error?.message}</AlertDescription>
    </Alert>
  )
}

export function PageLoader() {
  return (
    <div className="flex min-h-svh items-center justify-center text-muted-foreground">
      <Loader2Icon className="size-6 animate-spin" />
    </div>
  )
}
