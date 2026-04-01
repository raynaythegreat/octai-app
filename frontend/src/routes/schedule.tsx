import { createFileRoute } from "@tanstack/react-router"

import { CronPage } from "@/components/cron/CronPage"

export const Route = createFileRoute("/schedule")({
  component: CronPage,
})
