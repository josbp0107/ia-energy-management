import { useMutation } from "@tanstack/react-query"
import { AlertCircleIcon, Loader2Icon, ZapIcon } from "lucide-react"
import { useState, type FormEvent } from "react"
import { Navigate, useLocation, useNavigate } from "react-router"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth-context"

export function LoginPage() {
  const { user, login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from =
    (location.state as { from?: string } | null)?.from ?? "/dashboard"

  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")

  const mutation = useMutation({
    mutationFn: () => login(email, password),
    onSuccess: () => navigate(from, { replace: true }),
  })

  if (user) return <Navigate to={from} replace />

  function handleSubmit(event: FormEvent) {
    event.preventDefault()
    mutation.mutate()
  }

  const error = mutation.error
  const errorMessage =
    error instanceof ApiError && error.status === 401
      ? "Correo o contraseña incorrectos."
      : error?.message

  return (
    <div className="flex min-h-svh items-center justify-center bg-muted/40 p-4">
      <div className="flex w-full max-w-sm flex-col gap-6">
        <div className="flex items-center justify-center gap-2 font-semibold">
          <div className="flex size-8 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <ZapIcon className="size-4" />
          </div>
          AI Energy Management
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-xl">Iniciar sesión</CardTitle>
            <CardDescription>
              Monitoreo de medidores y anomalías detectadas por IA.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <Label htmlFor="email">Correo</Label>
                <Input
                  id="email"
                  type="email"
                  autoComplete="username"
                  placeholder="demo@bia.app"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                />
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="password">Contraseña</Label>
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </div>

              {errorMessage && (
                <Alert variant="destructive">
                  <AlertCircleIcon />
                  <AlertDescription>{errorMessage}</AlertDescription>
                </Alert>
              )}

              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending && <Loader2Icon className="animate-spin" />}
                Entrar
              </Button>
            </form>
          </CardContent>
        </Card>

        <p className="text-center text-xs text-muted-foreground">
          Usuario demo configurado en <code>backend/.env</code> (DEMO_EMAIL /
          DEMO_PASSWORD).
        </p>
      </div>
    </div>
  )
}
