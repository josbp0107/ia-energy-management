import { Outlet, useLocation } from "react-router"

import { AppSidebar } from "@/components/app-sidebar"
import { RunAnalysis } from "@/components/run-analysis"
import { Separator } from "@/components/ui/separator"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { navItems } from "@/lib/navigation"

export function AppLayout() {
  const { pathname } = useLocation()
  const section = navItems.find((item) => pathname.startsWith(item.url))

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
          <SidebarTrigger className="-ml-1" />
          <Separator orientation="vertical" className="mr-2 h-4" />
          <span className="text-sm font-medium">{section?.title}</span>
          <RunAnalysis />
        </header>
        <main className="flex flex-1 flex-col gap-6 p-4 md:p-6">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  )
}
