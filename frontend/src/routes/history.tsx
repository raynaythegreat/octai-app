import { createFileRoute } from "@tanstack/react-router"
import { HistoryPage } from "@/components/chat/history-page"

export const Route = createFileRoute("/history")({
  component: HistoryPage,
})
