import { createFileRoute } from "@tanstack/react-router"

import { TeamsPage } from "@/components/teams/TeamsPage"

export const Route = createFileRoute("/teams")({
  component: TeamsPage,
})
