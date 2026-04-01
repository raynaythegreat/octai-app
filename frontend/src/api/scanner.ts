// Scanner API — AI-powered URL analysis for discovering integrable items

export type DiscoveredItemType = "mcp_server" | "skill" | "tool" | "plugin" | "connection" | "reference_url" | "other"

export interface DiscoveredItem {
  type: DiscoveredItemType
  name: string
  description: string
  config?: Record<string, unknown>
}

export interface AnalyzeResult {
  items: DiscoveredItem[]
  url: string
  url_type: "github" | "website"
}

export interface IntegrateResultItem {
  name: string
  type: string
  success: boolean
  error?: string
}

export async function analyzeURL(url: string, crawlDepth?: number, maxPages?: number, sameDomain?: boolean): Promise<AnalyzeResult> {
  const res = await fetch("/api/scanner/analyze", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url, crawlDepth, maxPages, sameDomain }),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `Analysis failed: ${res.status}`)
  }
  return res.json()
}

export async function integrateItems(items: DiscoveredItem[]): Promise<IntegrateResultItem[]> {
  const res = await fetch("/api/scanner/integrate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(items),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `Integration failed: ${res.status}`)
  }
  return res.json()
}
