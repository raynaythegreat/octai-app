import { createFileRoute } from "@tanstack/react-router"

import { TailscaleVpnPage } from "@/components/settings/tailscale-vpn"

export const Route = createFileRoute("/settings/tailscale-vpn")({
  component: TailscaleVpnRoute,
})

function TailscaleVpnRoute() {
  return <TailscaleVpnPage />
}
