import { createFileRoute } from "@tanstack/react-router"

import { MobileAccessPage } from "@/components/settings/mobile-access"

export const Route = createFileRoute("/settings/mobile-access")({
  component: MobileAccessRoute,
})

function MobileAccessRoute() {
  return <MobileAccessPage />
}
