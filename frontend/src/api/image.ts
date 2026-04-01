export async function generateImage(params: {
  prompt: string
  model_index?: number
  size?: string
  quality?: string
}): Promise<{ url?: string; b64_json?: string; revised_prompt?: string; error?: string }> {
  const res = await fetch("/api/image/generate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  })
  return res.json() as Promise<{ url?: string; b64_json?: string; revised_prompt?: string; error?: string }>
}
