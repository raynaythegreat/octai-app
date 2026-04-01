import {
  IconEdit,
  IconKey,
  IconLoader2,
  IconStar,
  IconStarFilled,
  IconTrash,
} from "@tabler/icons-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"

import type { ModelInfo } from "@/api/models"
import { testModelKey } from "@/api/models"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"

interface ModelCardProps {
  model: ModelInfo
  onEdit: (model: ModelInfo) => void
  onSetDefault: (model: ModelInfo) => void
  onDelete: (model: ModelInfo) => void
  onRotateKey: (model: ModelInfo) => void
  onToggleChat: (model: ModelInfo, enabled: boolean) => void
  settingDefault: boolean
  togglingChat: boolean
}

export function ModelCard({
  model,
  onEdit,
  onSetDefault,
  onDelete,
  onRotateKey,
  onToggleChat,
  settingDefault,
  togglingChat,
}: ModelCardProps) {
  const { t } = useTranslation()
  const isOAuth = model.auth_method === "oauth"
  const isAvailable = model.available ?? model.configured
  const runtimeUnavailable = model.configured && !isAvailable
  const canSetDefault =
    model.configured &&
    model.chat_enabled &&
    !model.is_default &&
    !model.is_virtual

  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<{
    success: boolean
    error?: string
    models?: string[]
    note?: string
  } | null>(null)

  const handleTest = async () => {
    setTesting(true)
    setTestResult(null)
    try {
      const result = await testModelKey(model.index)
      setTestResult(result)
    } catch (e) {
      setTestResult({
        success: false,
        error: e instanceof Error ? e.message : "Test failed",
      })
    } finally {
      setTesting(false)
    }
  }

  return (
    <div
      className={[
        "group/card hover:bg-muted/30 relative flex w-full max-w-[36rem] flex-col gap-3 justify-self-start rounded-xl border p-4 transition-colors hover:shadow-xs",
        isAvailable
          ? "border-border/60 bg-card"
          : runtimeUnavailable
            ? "border-amber-500/40 bg-amber-500/5"
          : "border-border/50 bg-card/60",
      ].join(" ")}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <span
            className={[
              "mt-0.5 h-2 w-2 shrink-0 rounded-full",
              model.is_default
                ? "bg-green-400 shadow-[0_0_0_2px_rgba(74,222,128,0.35)]"
                : isAvailable
                  ? "bg-green-500"
                  : runtimeUnavailable
                    ? "bg-amber-500"
                  : "bg-muted-foreground/25",
            ].join(" ")}
            title={
              isAvailable
                ? t("models.status.configured")
                : runtimeUnavailable
                  ? t("models.status.runtimeUnavailable", {
                      defaultValue: "Configured, but not reachable right now",
                    })
                : t("models.status.unconfigured")
            }
          />
          <span className="text-foreground truncate text-sm font-semibold">
            {model.model_name}
          </span>
          {model.is_default && (
            <span className="bg-primary/10 text-primary shrink-0 rounded px-1.5 py-0.5 text-[10px] leading-none font-medium">
              {t("models.badge.default")}
            </span>
          )}
          {model.is_virtual && (
            <span className="bg-muted text-muted-foreground shrink-0 rounded px-1.5 py-0.5 text-[10px] leading-none font-medium">
              {t("models.badge.virtual")}
            </span>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-0.5">
          {model.is_default ? (
            <span
              className="text-primary p-1"
              title={t("models.badge.default")}
            >
              <IconStarFilled className="size-3.5" />
            </span>
          ) : (
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => onSetDefault(model)}
              disabled={settingDefault || !canSetDefault}
              title={t("models.action.setDefault")}
            >
              {settingDefault ? (
                <IconLoader2 className="size-3.5 animate-spin" />
              ) : (
                <IconStar className="size-3.5" />
              )}
            </Button>
          )}

          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onEdit(model)}
            title={t("models.action.edit")}
          >
            <IconEdit className="size-3.5" />
          </Button>

          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onRotateKey(model)}
            title={t("models.rotateKey.title", {
              name: model.model_name,
              defaultValue: "Rotate API key",
            })}
          >
            <IconKey className="size-3.5" />
          </Button>

          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onDelete(model)}
            disabled={model.is_default}
            title={t("models.action.delete")}
            className="text-muted-foreground hover:text-destructive hover:bg-destructive/10"
          >
            <IconTrash className="size-3.5" />
          </Button>
        </div>
      </div>

      <p className="text-muted-foreground truncate font-mono text-xs leading-snug">
        {model.model}
      </p>

      <div className="flex items-center gap-2">
        {isOAuth ? (
          <span className="text-muted-foreground bg-muted rounded px-1.5 py-0.5 text-[10px] font-medium">
            OAuth
          </span>
        ) : runtimeUnavailable ? (
          <span className="rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 dark:text-amber-300">
            {t("models.status.runtimeUnavailableShort", {
              defaultValue: "Runtime offline",
            })}
          </span>
        ) : model.configured && model.api_key ? (
          <span className="text-muted-foreground/70 flex items-center gap-1 font-mono text-[11px]">
            <IconKey className="size-3" />
            {model.api_key}
          </span>
        ) : (
          <span className="text-muted-foreground/50 text-[11px]">
            {t("models.status.unconfigured")}
          </span>
        )}

        <Button
          variant="ghost"
          size="sm"
          className="text-muted-foreground hover:text-foreground ml-auto h-6 px-2 text-[11px]"
          onClick={handleTest}
          disabled={testing}
        >
          {testing && <IconLoader2 className="size-3 animate-spin" />}
          {t("models.action.test", { defaultValue: "Test" })}
        </Button>
      </div>

      <div className="border-border/50 flex items-center justify-between gap-3 border-t pt-3">
        <div className="min-w-0">
          <p className="text-sm font-medium">
            {t("models.chatVisibility.label", {
              defaultValue: "Show in chat",
            })}
          </p>
          <p className="text-muted-foreground text-xs">
            {t("models.chatVisibility.description", {
              defaultValue:
                "Only chat-enabled and configured models appear in the chat menu.",
            })}
          </p>
        </div>
        <Switch
          checked={model.chat_enabled}
          onCheckedChange={(enabled) => onToggleChat(model, enabled)}
          disabled={togglingChat || model.is_virtual}
          aria-label={t("models.chatVisibility.label", {
            defaultValue: "Show in chat",
          })}
        />
      </div>

      {testResult && (
        <div
          className={`rounded-md px-2.5 py-1.5 text-xs ${testResult.success ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-400" : "bg-destructive/10 text-destructive"}`}
        >
          {testResult.success ? (
            <>
              <span className="font-medium">
                {t("models.test.success", { defaultValue: "Connected" })}
              </span>
              {testResult.models && testResult.models.length > 0 && (
                <span className="ml-1 opacity-75">
                  &middot;{" "}
                  {t("models.test.modelsCount", {
                    count: testResult.models.length,
                    defaultValue: `${testResult.models.length} models available`,
                  })}
                </span>
              )}
              {testResult.note && (
                <span className="ml-1 opacity-75">
                  &middot; {testResult.note}
                </span>
              )}
            </>
          ) : (
            testResult.error ?? t("models.test.failed", { defaultValue: "Failed" })
          )}
        </div>
      )}
    </div>
  )
}
