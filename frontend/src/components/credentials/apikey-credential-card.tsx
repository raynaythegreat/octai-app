import {
  IconCheck,
  IconLoader2,
  IconPlugConnected,
  IconRefresh,
  IconTrash,
  IconWifi,
} from "@tabler/icons-react"
import * as React from "react"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { type ModelInfo } from "@/api/models"
import { cn } from "@/lib/utils"

interface ApiKeyCredentialCardProps {
  providerName: string
  models: ModelInfo[]
  onSaveKey: (index: number, key: string) => Promise<void>
  onDeleteKey: (index: number) => Promise<void>
  onTestKey: (index: number) => Promise<{ success: boolean; models?: string[] }>
  onUpdateModels?: (providerModels: ModelInfo[]) => Promise<void>
}

const PROVIDER_DISPLAY_NAMES: Record<string, string> = {
  gemini: "Google Gemini",
  groq: "Groq",
  deepseek: "DeepSeek",
  mistral: "Mistral AI",
  openrouter: "OpenRouter",
  grok: "xAI Grok",
  cerebras: "Cerebras",
  perplexity: "Perplexity",
  together: "Together AI",
  moonshot: "Moonshot / Kimi",
  nvidia: "NVIDIA",
  zhipu: "Zhipu AI",
  qwen: "Qwen (Alibaba)",
  volcengine: "Volcengine",
  vivgrid: "Vivgrid",
  avian: "Avian",
  minimax: "MiniMax",
  longcat: "LongCat",
  modelscope: "ModelScope",
  azure: "Azure OpenAI",
  litellm: "LiteLLM (proxy)",
  vllm: "VLLM (local)",
  ollama: "Ollama (local)",
  github: "GitHub Copilot",
}

const LOCAL_PROVIDERS = new Set(["ollama", "vllm", "github"])

function isLocalProvider(providerName: string, models: ModelInfo[]): boolean {
  if (LOCAL_PROVIDERS.has(providerName)) return true
  const apiBase = models[0]?.api_base ?? ""
  return apiBase.includes("localhost") || apiBase.includes("127.0.0.1")
}

export function ApiKeyCredentialCard({
  providerName,
  models,
  onSaveKey,
  onDeleteKey,
  onTestKey,
  onUpdateModels,
}: ApiKeyCredentialCardProps) {
  const { t } = useTranslation()

  const primaryModel = models[0]
  const isConfigured = models.some((m) => m.configured)
  const isLocal = isLocalProvider(providerName, models)
  const displayName =
    PROVIDER_DISPLAY_NAMES[providerName] ??
    providerName.charAt(0).toUpperCase() + providerName.slice(1)

  const [keyInput, setKeyInput] = React.useState("")
  const [isSaving, setIsSaving] = React.useState(false)
  const [isTesting, setIsTesting] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [isUpdating, setIsUpdating] = React.useState(false)
  const [testPassed, setTestPassed] = React.useState(false)

  const handleSave = async () => {
    if (!keyInput.trim() || !primaryModel) return
    setIsSaving(true)
    try {
      await onSaveKey(primaryModel.index, keyInput.trim())
      setKeyInput("")
      setTestPassed(false)
    } finally {
      setIsSaving(false)
    }
  }

  const handleTest = async () => {
    if (!primaryModel) return
    setIsTesting(true)
    setTestPassed(false)
    try {
      const result = await onTestKey(primaryModel.index)
      setTestPassed(result.success)
    } finally {
      setIsTesting(false)
    }
  }

  const handleDelete = async () => {
    if (!primaryModel) return
    setIsDeleting(true)
    try {
      await onDeleteKey(primaryModel.index)
      setKeyInput("")
      setTestPassed(false)
    } finally {
      setIsDeleting(false)
    }
  }

  const handleUpdateModels = async () => {
    if (!onUpdateModels) return
    setIsUpdating(true)
    try {
      await onUpdateModels(models)
    } finally {
      setIsUpdating(false)
    }
  }

  const isAnyLoading = isSaving || isTesting || isDeleting || isUpdating

  if (isLocal) {
    return (
      <div className="bg-card border-border/60 flex flex-col gap-3 rounded-xl border p-4 shadow-sm">
        <div className="flex items-start justify-between">
          <div>
            <h3 className="text-sm font-semibold">{displayName}</h3>
            <p className="text-muted-foreground mt-0.5 text-xs">
              {models.length === 1
                ? models[0].model_name
                : `${models.length} models`}
            </p>
          </div>
          <Badge
            variant="outline"
            className={cn(
              "shrink-0 text-xs",
              isConfigured
                ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                : "border-border/60 text-muted-foreground",
            )}
          >
            {isConfigured
              ? t("credentials.local.connected")
              : t("credentials.local.notRunning")}
          </Badge>
        </div>

        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <IconPlugConnected className="size-4 shrink-0" />
          <span>
            {isConfigured
              ? t("credentials.local.connectedDesc")
              : t("credentials.local.notRunningDesc")}
          </span>
        </div>

        {onUpdateModels && (
          <Button
            size="sm"
            variant="ghost"
            className="text-muted-foreground h-7 w-full px-3 text-xs"
            onClick={handleUpdateModels}
            disabled={isAnyLoading}
          >
            {isUpdating ? (
              <IconLoader2 className="mr-1 size-3 animate-spin" />
            ) : (
              <IconRefresh className="mr-1 size-3" />
            )}
            {t("credentials.actions.updateModels")}
          </Button>
        )}
      </div>
    )
  }

  return (
    <div className="bg-card border-border/60 flex flex-col gap-3 rounded-xl border p-4 shadow-sm">
      <div className="flex items-start justify-between">
        <div>
          <h3 className="text-sm font-semibold">{displayName}</h3>
          <p className="text-muted-foreground mt-0.5 text-xs">
            {models.length === 1
              ? models[0].model_name
              : `${models.length} models`}
          </p>
        </div>
        <Badge
          variant="outline"
          className={cn(
            "shrink-0 text-xs",
            isConfigured
              ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
              : "border-border/60 text-muted-foreground",
          )}
        >
          {isConfigured
            ? t("credentials.apikey.configured")
            : t("credentials.apikey.notConfigured")}
        </Badge>
      </div>

      <Input
        type="password"
        value={keyInput}
        onChange={(e) => setKeyInput(e.target.value)}
        placeholder={
          isConfigured
            ? "••••••••••••••••"
            : t("credentials.apikey.keyPlaceholder")
        }
        className="h-8 text-xs font-mono"
        onKeyDown={(e) => {
          if (e.key === "Enter" && keyInput.trim()) {
            void handleSave()
          }
        }}
      />

      <div className="flex flex-wrap items-center gap-1.5">
        <Button
          size="sm"
          variant="default"
          className="h-7 px-3 text-xs"
          onClick={handleSave}
          disabled={isAnyLoading || !keyInput.trim()}
        >
          {isSaving ? (
            <IconLoader2 className="mr-1 size-3 animate-spin" />
          ) : null}
          {t("common.save")}
        </Button>

        <Button
          size="sm"
          variant="outline"
          className={cn(
            "h-7 px-3 text-xs",
            testPassed
              ? "border-emerald-500/40 text-emerald-600 dark:text-emerald-400"
              : "",
          )}
          onClick={handleTest}
          disabled={isAnyLoading || !isConfigured}
        >
          {isTesting ? (
            <IconLoader2 className="mr-1 size-3 animate-spin" />
          ) : testPassed ? (
            <IconCheck className="mr-1 size-3" />
          ) : (
            <IconWifi className="mr-1 size-3" />
          )}
          {t("credentials.actions.test")}
        </Button>

        {onUpdateModels && isConfigured && (
          <Button
            size="sm"
            variant="ghost"
            className="text-muted-foreground h-7 px-3 text-xs"
            onClick={handleUpdateModels}
            disabled={isAnyLoading}
          >
            {isUpdating ? (
              <IconLoader2 className="mr-1 size-3 animate-spin" />
            ) : (
              <IconRefresh className="mr-1 size-3" />
            )}
            {t("credentials.actions.updateModels")}
          </Button>
        )}

        <Button
          size="sm"
          variant="ghost"
          className="text-muted-foreground hover:text-destructive ml-auto h-7 px-2 text-xs"
          onClick={handleDelete}
          disabled={isAnyLoading || !isConfigured}
          title={t("credentials.actions.deleteKey")}
        >
          {isDeleting ? (
            <IconLoader2 className="size-3 animate-spin" />
          ) : (
            <IconTrash className="size-3" />
          )}
        </Button>
      </div>
    </div>
  )
}
