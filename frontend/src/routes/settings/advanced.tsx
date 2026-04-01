import { createFileRoute } from "@tanstack/react-router"

import { AdvancedPage } from "@/components/settings/advanced"

export const Route = createFileRoute("/settings/advanced")({
  component: AdvancedRoute,
})

function AdvancedRoute() {
  return <AdvancedPage />
}
