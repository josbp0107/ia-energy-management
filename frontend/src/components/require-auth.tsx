import { Navigate, Outlet, useLocation } from "react-router"

import { useAuth } from "@/lib/auth-context"

export function RequireAuth() {
  const { user } = useAuth()
  const location = useLocation()

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }
  return <Outlet />
}
