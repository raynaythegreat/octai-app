// Sessions API — list and retrieve chat session history

export interface SessionSummary {
  id: string
  title: string
  preview: string
  message_count: number
  created: string
  updated: string
  channel?: string
}

export interface SessionDetail {
  id: string
  messages: { role: "user" | "assistant"; content: string }[]
  summary: string
  created: string
  updated: string
  channel?: string
}

export async function getSessions(
  offset: number = 0,
  limit: number = 20,
  channel?: string,
): Promise<SessionSummary[]> {
  const params = new URLSearchParams({
    offset: offset.toString(),
    limit: limit.toString(),
  })
  if (channel) {
    params.set("channel", channel)
  }

  const res = await fetch(`/api/sessions?${params.toString()}`)
  if (!res.ok) {
    throw new Error(`Failed to fetch sessions: ${res.status}`)
  }
  return res.json()
}

export async function getSessionHistory(id: string, channel?: string): Promise<SessionDetail> {
  const url = channel
    ? `/api/sessions/${encodeURIComponent(id)}?channel=${encodeURIComponent(channel)}`
    : `/api/sessions/${encodeURIComponent(id)}`
  const res = await fetch(url)
  if (!res.ok) {
    throw new Error(`Failed to fetch session ${id}: ${res.status}`)
  }
  return res.json()
}

export async function deleteSession(id: string, channel?: string): Promise<void> {
  const url = channel
    ? `/api/sessions/${encodeURIComponent(id)}?channel=${encodeURIComponent(channel)}`
    : `/api/sessions/${encodeURIComponent(id)}`
  const res = await fetch(url, {
    method: "DELETE",
  })
  if (!res.ok) {
    throw new Error(`Failed to delete session ${id}: ${res.status}`)
  }
}
