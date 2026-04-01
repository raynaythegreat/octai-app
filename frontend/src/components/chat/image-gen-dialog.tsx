import { IconLoader2, IconPhoto, IconX } from "@tabler/icons-react"
import { Dialog as DialogPrimitive } from "radix-ui"
import * as React from "react"

import { generateImage } from "@/api/image"
import { getImageModels, type ModelInfo } from "@/api/models"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface ImageGenDialogProps {
  open: boolean
  onClose: () => void
  onInsert: (dataUrl: string, prompt: string) => void
}

const SIZES = ["1024x1024", "1792x1024", "1024x1792"] as const
const QUALITIES = ["standard", "hd"] as const

export function ImageGenDialog({ open, onClose, onInsert }: ImageGenDialogProps) {
  const [prompt, setPrompt] = React.useState("")
  const [size, setSize] = React.useState<string>("1024x1024")
  const [quality, setQuality] = React.useState<string>("standard")
  const [modelIndex, setModelIndex] = React.useState(0)
  const [imageModels, setImageModels] = React.useState<ModelInfo[]>([])
  const [isLoading, setIsLoading] = React.useState(false)
  const [resultDataUrl, setResultDataUrl] = React.useState<string | null>(null)
  const [resultPrompt, setResultPrompt] = React.useState<string>("")
  const [error, setError] = React.useState<string | null>(null)

  // Load image models when dialog opens
  React.useEffect(() => {
    if (!open) return
    getImageModels()
      .then((res) =>
        setImageModels(
          res.models.filter((model) => model.chat_enabled && model.configured),
        ),
      )
      .catch(() => setImageModels([]))
  }, [open])

  React.useEffect(() => {
    if (modelIndex >= imageModels.length) {
      setModelIndex(0)
    }
  }, [imageModels.length, modelIndex])

  // Reset state when dialog closes
  React.useEffect(() => {
    if (!open) {
      setPrompt("")
      setResultDataUrl(null)
      setResultPrompt("")
      setError(null)
      setIsLoading(false)
    }
  }, [open])

  const handleGenerate = async () => {
    if (!prompt.trim()) return
    setIsLoading(true)
    setResultDataUrl(null)
    setError(null)

    try {
      const res = await generateImage({
        prompt: prompt.trim(),
        model_index: modelIndex,
        size,
        quality,
      })

      if (res.error) {
        setError(res.error)
        return
      }

      if (res.b64_json) {
        const dataUrl = `data:image/png;base64,${res.b64_json}`
        setResultDataUrl(dataUrl)
        setResultPrompt(res.revised_prompt ?? prompt)
      } else if (res.url) {
        // Fetch the URL and convert to data URL so it can be attached
        try {
          const imgRes = await fetch(res.url)
          const blob = await imgRes.blob()
          const reader = new FileReader()
          reader.onload = (e) => {
            setResultDataUrl(e.target?.result as string)
          }
          reader.readAsDataURL(blob)
          setResultPrompt(res.revised_prompt ?? prompt)
        } catch {
          // Fall back to using the URL directly as a placeholder data URL
          setResultDataUrl(res.url)
          setResultPrompt(res.revised_prompt ?? prompt)
        }
      } else {
        setError("No image returned from the model.")
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to generate image")
    } finally {
      setIsLoading(false)
    }
  }

  const handleInsert = () => {
    if (resultDataUrl) {
      onInsert(resultDataUrl, resultPrompt || prompt)
    }
  }

  return (
    <DialogPrimitive.Root open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay
          className="fixed inset-0 z-50 bg-black/30 supports-backdrop-filter:backdrop-blur-sm data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0"
        />
        <DialogPrimitive.Content
          className={cn(
            "fixed top-1/2 left-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2",
            "bg-background rounded-2xl border border-border/80 shadow-xl p-6",
            "data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95",
            "data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95",
          )}
        >
          {/* Header */}
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <IconPhoto className="size-5 text-violet-400" />
              <DialogPrimitive.Title className="text-base font-semibold">
                Generate Image
              </DialogPrimitive.Title>
            </div>
            <DialogPrimitive.Close asChild>
              <Button variant="ghost" size="icon" className="size-7 rounded-full">
                <IconX className="size-4" />
                <span className="sr-only">Close</span>
              </Button>
            </DialogPrimitive.Close>
          </div>

          {/* Prompt */}
          <div className="mb-4">
            <label className="mb-1.5 block text-sm font-medium">Prompt</label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Describe the image you want to generate…"
              rows={3}
              className={cn(
                "w-full resize-none rounded-xl border border-border/80 bg-muted/40 px-3 py-2 text-sm",
                "placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-violet-500/40",
              )}
              onKeyDown={(e) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                  e.preventDefault()
                  void handleGenerate()
                }
              }}
            />
          </div>

          {/* Controls row */}
          <div className="mb-4 flex flex-wrap gap-3">
            {/* Size selector */}
            <div className="flex-1 min-w-[140px]">
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">Size</label>
              <select
                value={size}
                onChange={(e) => setSize(e.target.value)}
                className="w-full rounded-lg border border-border/80 bg-muted/40 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500/40"
              >
                {SIZES.map((s) => (
                  <option key={s} value={s}>{s}</option>
                ))}
              </select>
            </div>

            {/* Quality toggle */}
            <div className="flex-1 min-w-[120px]">
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">Quality</label>
              <div className="flex rounded-lg border border-border/80 overflow-hidden">
                {QUALITIES.map((q) => (
                  <button
                    key={q}
                    type="button"
                    onClick={() => setQuality(q)}
                    className={cn(
                      "flex-1 py-1.5 text-sm capitalize transition-colors",
                      quality === q
                        ? "bg-violet-500 text-white"
                        : "bg-muted/40 text-muted-foreground hover:bg-muted/70",
                    )}
                  >
                    {q}
                  </button>
                ))}
              </div>
            </div>

            {/* Model selector (only shown when models are available) */}
            {imageModels.length > 0 && (
              <div className="flex-1 min-w-[140px]">
                <label className="mb-1.5 block text-xs font-medium text-muted-foreground">Model</label>
                <select
                  value={modelIndex}
                  onChange={(e) => setModelIndex(Number(e.target.value))}
                  className="w-full rounded-lg border border-border/80 bg-muted/40 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500/40"
                >
                  {imageModels.map((m, i) => (
                    <option key={i} value={i}>{m.model_name || m.model}</option>
                  ))}
                </select>
              </div>
            )}
          </div>

          {/* Error message */}
          {error && (
            <div className="mb-4 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          )}

          {!error && imageModels.length === 0 && (
            <div className="mb-4 rounded-lg bg-amber-500/10 px-3 py-2 text-sm text-amber-600 dark:text-amber-400">
              Enable and configure at least one image model on the Models page to use image generation in chat.
            </div>
          )}

          {/* Result image */}
          {resultDataUrl && !isLoading && (
            <div className="mb-4">
              <img
                src={resultDataUrl}
                alt={resultPrompt || prompt}
                className="w-full rounded-xl border border-border/80 object-contain max-h-64"
              />
              {resultPrompt && resultPrompt !== prompt && (
                <p className="mt-1.5 text-xs text-muted-foreground line-clamp-2">
                  <span className="font-medium">Revised:</span> {resultPrompt}
                </p>
              )}
            </div>
          )}

          {/* Loading state */}
          {isLoading && (
            <div className="mb-4 flex h-32 items-center justify-center rounded-xl border border-border/80 bg-muted/20">
              <IconLoader2 className="size-6 animate-spin text-violet-400" />
            </div>
          )}

          {/* Footer actions */}
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={onClose} disabled={isLoading}>
              Cancel
            </Button>
            {resultDataUrl && !isLoading ? (
              <>
                <Button
                  variant="outline"
                  onClick={() => { setResultDataUrl(null); setError(null) }}
                >
                  Regenerate
                </Button>
                <Button
                  className="bg-violet-500 text-white hover:bg-violet-600"
                  onClick={handleInsert}
                >
                  Insert into Chat
                </Button>
              </>
            ) : (
              <Button
                className="bg-violet-500 text-white hover:bg-violet-600"
                onClick={() => void handleGenerate()}
                disabled={!prompt.trim() || isLoading || imageModels.length === 0}
              >
                {isLoading ? (
                  <>
                    <IconLoader2 className="size-4 animate-spin mr-2" />
                    Generating…
                  </>
                ) : (
                  "Generate"
                )}
              </Button>
            )}
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}
