import { createFileRoute } from "@tanstack/react-router"

import { PluginsPage } from "@/components/agent/plugins-page"

export const Route = createFileRoute("/agent/plugins")({
  component: PluginsPage,
})
