export interface MobileAccessConfig {
  enabled: boolean
  connectionMethod: "tailscale" | "lan"
  accessUrl: string
}

export interface ConnectedDevice {
  id: string
  name: string
  type: string
  ip: string
  lastConnected: string
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    let message = `API error: ${res.status} ${res.statusText}`
    try {
      const body = (await res.json()) as {
        error?: string
        errors?: string[]
      }
      if (Array.isArray(body.errors) && body.errors.length > 0) {
        message = body.errors.join("; ")
      } else if (typeof body.error === "string" && body.error.trim() !== "") {
        message = body.error
      }
    } catch {
      // Keep fallback error message when response body is not JSON.
    }
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

export async function getMobileAccessStatus(): Promise<MobileAccessConfig> {
  return request<MobileAccessConfig>("/api/mobile/status")
}

export async function setMobileAccessConfig(
  config: Partial<MobileAccessConfig>,
): Promise<void> {
  await request<void>("/api/mobile/config", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  })
}

export async function getConnectedDevices(): Promise<ConnectedDevice[]> {
  return request<ConnectedDevice[]>("/api/mobile/devices")
}
