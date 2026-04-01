import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { cn } from "@/lib/utils"

interface ThinkingIndicatorProps {
  className?: string
  startTime?: number
}

export function ThinkingIndicator({ className, startTime = Date.now() }: ThinkingIndicatorProps) {
  const { t } = useTranslation()
  const [elapsedTime, setElapsedTime] = useState(0)

  useEffect(() => {
    const interval = setInterval(() => {
      setElapsedTime(Math.floor((Date.now() - startTime) / 1000))
    }, 1000)
    return () => clearInterval(interval)
  }, [startTime])

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, "0")}`
  }

  return (
    <div className={cn("flex w-full flex-col gap-1.5", className)}>
      <div className="text-muted-foreground flex items-center gap-2 px-1 text-xs opacity-70">
        <span>OctAi</span>
      </div>
      <div className="bg-card inline-flex w-fit max-w-xs flex-col gap-3 rounded-xl border px-5 py-4">
        {/* Pulsing dots */}
        <div className="flex items-center gap-1.5">
          <span className="size-2 animate-bounce rounded-full bg-violet-400/70 [animation-delay:-0.3s]" />
          <span className="size-2 animate-bounce rounded-full bg-violet-400/70 [animation-delay:-0.15s]" />
          <span className="size-2 animate-bounce rounded-full bg-violet-400/70" />
        </div>

        {/* Progress bar */}
        <div className="bg-muted relative h-1 w-36 overflow-hidden rounded-full">
          <div className="absolute inset-0 animate-[shimmer_2s_infinite] rounded-full bg-gradient-to-r from-violet-500/60 via-violet-400/80 to-violet-500/60 bg-[length:200%_100%]" />
        </div>

        {/* Thinking text with timer */}
        <div className="flex items-center gap-2">
          <p className="text-muted-foreground text-xs font-medium">
            {t("chat.thinking.label", "Thinking...")}
          </p>
          <span className="text-muted-foreground/60 text-xs font-mono">
            {formatTime(elapsedTime)}
          </span>
        </div>
      </div>
    </div>
  )
}
