import { createContext, useContext } from "react"

import type { User } from "@/lib/types"

export interface AuthState {
  user: User | null
  login: (email: string, password: string) => Promise<void>
  logout: () => void
}

export const AuthContext = createContext<AuthState | null>(null)

export function useAuth() {
  const auth = useContext(AuthContext)
  if (!auth) throw new Error("useAuth debe usarse dentro de <AuthProvider>")
  return auth
}
