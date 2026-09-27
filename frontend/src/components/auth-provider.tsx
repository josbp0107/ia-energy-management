import { useQueryClient } from "@tanstack/react-query"
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react"
import { toast } from "sonner"

import { api, setUnauthorizedHandler } from "@/lib/api"
import { AuthContext, type AuthState } from "@/lib/auth-context"
import {
  clearSession,
  loadSession,
  saveSession,
  type Session,
} from "@/lib/session"

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [session, setSession] = useState<Session | null>(loadSession)

  const login = useCallback(async (email: string, password: string) => {
    const { token, user } = await api.login(email, password)
    const next = { token, user }
    saveSession(next)
    setSession(next)
    toast.dismiss("session-expired")
  }, [])

  const logout = useCallback(() => {
    clearSession()
    setSession(null)
    queryClient.clear() // limpia cache
  }, [queryClient])

  useEffect(() => {
    setUnauthorizedHandler(() => {
      setSession(null)
      queryClient.clear()
      toast.warning("Tu sesión expiró. Vuelve a iniciar sesión.", {
        id: "session-expired",
      })
    })
    return () => setUnauthorizedHandler(null)
  }, [queryClient])

  const value = useMemo<AuthState>(
    () => ({ user: session?.user ?? null, login, logout }),
    [session, login, logout]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
