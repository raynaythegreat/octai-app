import { createFileRoute } from "@tanstack/react-router"

import { LoopsPage } from "@/components/loops/LoopsPage"

export const Route = createFileRoute("/loops")({
  component: LoopsPage,
})
