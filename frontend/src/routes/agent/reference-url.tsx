import { createFileRoute } from "@tanstack/react-router"

import { ReferenceUrlPage } from "@/components/agent/reference-url-page"

export const Route = createFileRoute("/agent/reference-url")({
  component: ReferenceUrlPage,
})
