import {
  IconPlugConnectedX,
  IconRobotOff,
  IconStar,
} from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"

interface ChatEmptyStateProps {
  hasSavedModels: boolean
  hasChatEnabledModels: boolean
  hasAvailableModels: boolean
  defaultModelName: string
  isAutoMode: boolean
  isConnected: boolean
}

export function ChatEmptyState({
  hasSavedModels,
  hasChatEnabledModels,
  hasAvailableModels,
  defaultModelName,
  isAutoMode,
  isConnected,
}: ChatEmptyStateProps) {
  const { t } = useTranslation()

  if (!hasSavedModels) {
    return (
      <div className="animate-in fade-in duration-500 flex h-full flex-col items-center justify-center px-4">
        <div className="mb-8 flex h-20 w-20 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-500 transition-transform duration-500 hover:scale-105">
          <IconRobotOff className="h-10 w-10" />
        </div>
        <h3 className="mb-3 text-2xl font-semibold tracking-tight">
          {t("chat.empty.noConfiguredModel")}
        </h3>
        <p className="text-muted-foreground mb-6 max-w-sm text-center text-base">
          {t("chat.empty.noConfiguredModelDescription")}
        </p>
        <Button asChild variant="outline" size="sm" className="px-6">
          <Link to="/models">{t("chat.empty.goToModels")}</Link>
        </Button>
      </div>
    )
  }

  if (!hasChatEnabledModels) {
    return (
      <div className="animate-in fade-in duration-500 flex h-full flex-col items-center justify-center px-4">
        <div className="mb-8 flex h-20 w-20 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-500 transition-transform duration-500 hover:scale-105">
          <IconStar className="h-10 w-10" />
        </div>
        <h3 className="mb-3 text-2xl font-semibold tracking-tight">
          {t("chat.empty.noSelectedModel")}
        </h3>
        <p className="text-muted-foreground mb-6 max-w-sm text-center text-base">
          {t("chat.empty.noChatEnabledModelDescription", {
            defaultValue:
              "Enable at least one model on the Models page to show it in chat.",
          })}
        </p>
        <Button asChild variant="outline" size="sm" className="px-6">
          <Link to="/models">{t("chat.empty.goToModels")}</Link>
        </Button>
      </div>
    )
  }

  if (!hasAvailableModels) {
    return (
      <div className="animate-in fade-in duration-500 flex h-full flex-col items-center justify-center px-4">
        <div className="mb-8 flex h-20 w-20 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-500 transition-transform duration-500 hover:scale-105">
          <IconRobotOff className="h-10 w-10" />
        </div>
        <h3 className="mb-3 text-2xl font-semibold tracking-tight">
          {t("chat.empty.noConfiguredModel")}
        </h3>
        <p className="text-muted-foreground mb-6 max-w-sm text-center text-base">
          {t("chat.empty.noChatReadyModelDescription", {
            defaultValue:
              "Your chat-enabled models are not configured yet. Add an API key, connect OAuth, or start the local runtime.",
          })}
        </p>
        <Button asChild variant="outline" size="sm" className="px-6">
          <Link to="/models">{t("chat.empty.goToModels")}</Link>
        </Button>
      </div>
    )
  }

  if (!defaultModelName && !isAutoMode) {
    return (
      <div className="animate-in fade-in duration-500 flex h-full flex-col items-center justify-center px-4">
        <div className="mb-8 flex h-20 w-20 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-500 transition-transform duration-500 hover:scale-105">
          <IconStar className="h-10 w-10" />
        </div>
        <h3 className="mb-3 text-2xl font-semibold tracking-tight">
          {t("chat.empty.noSelectedModel")}
        </h3>
        <p className="text-muted-foreground mb-6 max-w-sm text-center text-base">
          {t("chat.empty.noSelectedModelDescription", {
            defaultValue:
              "Choose one of your chat-enabled models from the chat model menu before sending a message.",
          })}
        </p>
      </div>
    )
  }

  if (!isConnected) {
    return (
      <div className="animate-in fade-in duration-500 flex h-full flex-col items-center justify-center px-4">
        <div className="mb-8 flex h-20 w-20 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-500 transition-transform duration-500 hover:scale-105">
          <IconPlugConnectedX className="h-10 w-10" />
        </div>
        <h3 className="mb-3 text-2xl font-semibold tracking-tight">
          {t("chat.empty.notRunning")}
        </h3>
        <p className="text-muted-foreground mb-6 max-w-sm text-center text-base">
          {t("chat.empty.notRunningDescription")}
        </p>
      </div>
    )
  }

  return (
    <div className="animate-in fade-in zoom-in-95 duration-500 flex h-full flex-col items-center justify-center px-4">
      <div className="mb-8 flex h-20 w-20 items-center justify-center rounded-2xl bg-violet-500/10 transition-all duration-500 hover:scale-105 hover:bg-violet-500/15">
        <img src="/favicon.svg" alt="OctAi" className="h-14 w-14" />
      </div>
      <h3 className="mb-3 text-2xl font-semibold tracking-tight">{t("chat.welcome")}</h3>
      <p className="text-muted-foreground max-w-sm text-center text-base">
        {t("chat.welcomeDesc")}
      </p>
    </div>
  )
}
