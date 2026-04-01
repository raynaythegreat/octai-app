import { createFileRoute } from "@tanstack/react-router"

import { ConfigPage } from "@/components/config/config-page"

export const Route = createFileRoute("/settings")({
  component: ConfigPage,
})
