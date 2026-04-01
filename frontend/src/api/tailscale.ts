export interface TailscaleStatus {
  installed: boolean
  connected: boolean
  ip: string
  hostname: string
  tailnetUrl: string
  magicDns: boolean
  autoStart: boolean
}

export interface TailscaleDevice {
  id: string
  hostname: string
  ip: string
  os: string
  online: boolean
  lastSeen: string
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

export async function getTailscaleStatus(): Promise<TailscaleStatus> {
  return request<TailscaleStatus>("/api/tailscale/status")
}

export async function installTailscale(): Promise<{
  success: boolean
  message: string
}> {
  return request<{ success: boolean; message: string }>(
    "/api/tailscale/install",
    { method: "POST" },
  )
}

export async function authenticateTailscale(): Promise<{ url: string }> {
  return request<{ url: string }>("/api/tailscale/authenticate", {
    method: "POST",
  })
}

export async function getTailnetDevices(): Promise<TailscaleDevice[]> {
  return request<TailscaleDevice[]>("/api/tailscale/devices")
}

export async function setTailscaleConfig(
  config: Partial<TailscaleStatus>,
): Promise<void> {
  await request<void>("/api/tailscale/config", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  })
}
