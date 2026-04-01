import { Navigate, Outlet, createFileRoute, useRouterState } from "@tanstack/react-router"

export const Route = createFileRoute("/config")({
  component: ConfigRouteLayout,
})

function ConfigRouteLayout() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  if (pathname === "/config") {
    return <Navigate to="/settings" />
  }

  return <Outlet />
}
