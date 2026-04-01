import { createFileRoute } from "@tanstack/react-router"

import { WorkflowsPage } from "@/components/workflow/WorkflowsPage"

export const Route = createFileRoute("/workflows")({
  component: WorkflowsPage,
})
