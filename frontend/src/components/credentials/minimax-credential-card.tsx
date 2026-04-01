import {
  IconClockHour4,
  IconLoader2,
  IconPlayerStopFilled,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { OAuthProviderStatus } from "@/api/oauth"
import { Button } from "@/components/ui/button"

import { CredentialCard } from "./credential-card"

interface MiniMaxCredentialCardProps {
  status?: OAuthProviderStatus
  activeAction: string
  onStartDeviceCode: () => void
  onStopLoading: () => void
  onAskLogout: () => void
}

export function MiniMaxCredentialCard({
  status,
  activeAction,
  onStartDeviceCode,
  onStopLoading,
  onAskLogout,
}: MiniMaxCredentialCardProps) {
  const { t } = useTranslation()
  const actionBusy = activeAction !== ""
  const deviceLoading = activeAction === "minimax:device"

  return (
    <CredentialCard
      title={
        <span className="inline-flex items-center gap-2">
          <span className="border-muted inline-flex size-6 items-center justify-center rounded-full border text-xs font-bold">
            M
          </span>
          <span>MiniMax</span>
        </span>
      }
      description={t("credentials.providers.minimax.description")}
      status={status?.status ?? "not_logged_in"}
      authMethod={status?.auth_method}
      details={
        status?.account_id ? (
          <p>
            {t("credentials.labels.account")}: {status.account_id}
          </p>
        ) : null
      }
      actions={
        <div className="border-muted flex h-[120px] flex-col justify-center rounded-lg border p-3">
          <div className="flex flex-wrap items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={actionBusy}
              onClick={onStartDeviceCode}
            >
              {deviceLoading && (
                <IconLoader2 className="size-4 animate-spin" />
              )}
              <IconClockHour4 className="size-4" />
              {t("credentials.actions.deviceCode")}
            </Button>

            {deviceLoading && (
              <Button
                size="icon-xs"
                variant="secondary"
                onClick={onStopLoading}
                className="text-destructive hover:bg-destructive/10 hover:text-destructive"
              >
                <IconPlayerStopFilled className="size-4" />
              </Button>
            )}
          </div>
        </div>
      }
      footer={
        status?.logged_in ? (
          <Button
            variant="ghost"
            size="sm"
            disabled={actionBusy}
            onClick={onAskLogout}
            className="text-destructive hover:bg-destructive/10 hover:text-destructive"
          >
            {activeAction === "minimax:logout" && (
              <IconLoader2 className="size-4 animate-spin" />
            )}
            {t("credentials.actions.logout")}
          </Button>
        ) : null
      }
    />
  )
}
