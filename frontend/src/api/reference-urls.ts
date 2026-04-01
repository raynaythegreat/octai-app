// Reference URLs API — save and categorize useful links for agents

export interface ReferenceURL {
  id: string
  url: string
  title: string
  description: string
  category: string
  tags: string[]
  notes?: string
  added_at: string
}

export interface ReferenceURLsResponse {
  references: ReferenceURL[]
}

export async function getReferenceURLs(): Promise<ReferenceURLsResponse> {
  const res = await fetch("/api/reference-urls")
  if (!res.ok) throw new Error(`Failed to load references: ${res.status}`)
  return res.json()
}

export async function addReferenceURL(url: string, notes?: string): Promise<ReferenceURL> {
  const res = await fetch("/api/reference-urls", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url, notes }),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `Failed to add reference: ${res.status}`)
  }
  return res.json()
}

export async function deleteReferenceURL(id: string): Promise<void> {
  const res = await fetch(`/api/reference-urls/${id}`, { method: "DELETE" })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `Failed to delete reference: ${res.status}`)
  }
}

export async function updateReferenceURL(
  id: string,
  updates: { notes?: string; category?: string },
): Promise<ReferenceURL> {
  const res = await fetch(`/api/reference-urls/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(updates),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `Failed to update reference: ${res.status}`)
  }
  return res.json()
}
