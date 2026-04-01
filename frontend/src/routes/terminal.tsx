import { createFileRoute } from "@tanstack/react-router"

import { TerminalTab } from "@/components/terminal/terminal-tab"

export const Route = createFileRoute("/terminal")({
  component: TerminalTab,
})
