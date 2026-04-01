import { createFileRoute } from "@tanstack/react-router"

import { AiUrlPage } from "@/components/agent/ai-url-page"

export const Route = createFileRoute("/agent/ai-url")({
  component: AiUrlPage,
})
