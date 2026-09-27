import { createBrowserRouter, Navigate } from "react-router"

import { AppLayout } from "@/components/app-layout"
import { PageLoader } from "@/components/page"
import { RequireAuth } from "@/components/require-auth"
import { LoginPage } from "@/pages/login-page"

const lazyPage =
  <K extends string>(
    load: () => Promise<Record<K, React.ComponentType>>,
    name: K
  ) =>
  async () => ({ Component: (await load())[name] })

export const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  {
    element: <RequireAuth />,
    hydrateFallbackElement: <PageLoader />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { path: "/", element: <Navigate to="/dashboard" replace /> },
          {
            path: "/dashboard",
            lazy: lazyPage(
              () => import("@/pages/dashboard-page"),
              "DashboardPage"
            ),
          },
          {
            path: "/meters",
            lazy: lazyPage(() => import("@/pages/meters-page"), "MetersPage"),
          },
          {
            path: "/meters/:meterId",
            lazy: lazyPage(
              () => import("@/pages/meter-detail-page"),
              "MeterDetailPage"
            ),
          },
          {
            path: "/anomalies",
            lazy: lazyPage(
              () => import("@/pages/anomalies-page"),
              "AnomaliesPage"
            ),
          },
          {
            path: "/anomalies/:id",
            lazy: lazyPage(
              () => import("@/pages/investigation/investigation-page"),
              "InvestigationPage"
            ),
          },
        ],
      },
    ],
  },
  { path: "*", element: <Navigate to="/dashboard" replace /> },
])
