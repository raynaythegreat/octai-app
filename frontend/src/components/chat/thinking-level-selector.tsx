import { IconBrain } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { ModelInfo } from "@/api/models"
import type { ThinkingLevel } from "@/features/chat/state"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface ThinkingLevelSelectorProps {
  models: ModelInfo[]
  defaultModelName: string
  isAutoMode: boolean
  thinkingLevel: ThinkingLevel
  onThinkingLevelChange: (level: ThinkingLevel) => void
}

const THINKING_LEVELS: { value: ThinkingLevel; label: string }[] = [
  { value: "off", label: "Off" },
  { value: "low", label: "Low" },
  { value: "medium", label: "Medium" },
  { value: "high", label: "High" },
  { value: "extra_high", label: "Extra High" },
]

// Reasoning-capable model identifiers
const REASONING_MODEL_PATTERNS = [
  /o1/i,
  /o3/i,
  /deepseek-reasoner/i,
  /grok-4-reasoning/i,
]

function isReasoningModel(modelName: string): boolean {
  return REASONING_MODEL_PATTERNS.some((pattern) => pattern.test(modelName))
}

function supportsThinking(model?: ModelInfo): boolean {
  if (!model) return false
  return isReasoningModel(model.model_name) || isReasoningModel(model.model)
}

export function ThinkingLevelSelector({
  models,
  defaultModelName,
  isAutoMode,
  thinkingLevel,
  onThinkingLevelChange,
}: ThinkingLevelSelectorProps) {
  const { t } = useTranslation()

  // Find the currently selected model
  const selectedModel = models.find((m) => m.model_name === defaultModelName)

  // Determine if thinking level should be shown
  // In auto mode, we don't know which model will be used, so we show it if any model supports thinking
  const shouldShowThinking = isAutoMode
    ? models.some(supportsThinking)
    : supportsThinking(selectedModel)

  if (!shouldShowThinking) {
    return null
  }

  return (
    <Select
      value={thinkingLevel}
      onValueChange={(value) => onThinkingLevelChange(value as ThinkingLevel)}
    >
      <SelectTrigger
        size="sm"
        className="text-muted-foreground hover:text-foreground focus-visible:border-input h-8 max-w-[140px] min-w-[80px] bg-transparent shadow-none focus-visible:ring-0"
      >
        <IconBrain className="mr-1.5 size-4 shrink-0 text-violet-400" />
        <SelectValue placeholder={t("chat.thinkingLevel")} />
      </SelectTrigger>
      <SelectContent position="popper" align="start">
        {THINKING_LEVELS.map((level) => (
          <SelectItem key={level.value} value={level.value}>
            {level.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
