import { IconLoader2, IconMessageCircle, IconTrash } from "@tabler/icons-react"
import dayjs from "dayjs"
import relativeTime from "dayjs/plugin/relativeTime"
import * as React from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "@tanstack/react-router"

import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { switchChatSession } from "@/features/chat/controller"
import { useSessionHistory } from "@/hooks/use-session-history"
import { usePicoChat } from "@/hooks/use-pico-chat"

dayjs.extend(relativeTime)

export function HistoryPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { activeSessionId, newChat } = usePicoChat()

  const {
    sessions,
    hasMore,
    loadError,
    loadErrorMessage,
    observerRef,
    loadSessions,
    handleDeleteSession,
  } = useSessionHistory({
    activeSessionId,
    onDeletedActiveSession: newChat,
  })

  React.useEffect(() => {
    void loadSessions(true)
  }, [loadSessions])

  const handleSwitchSession = async (sessionId: string) => {
    await switchChatSession(sessionId)
    void navigate({ to: "/" })
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("chat.history")} />

      <div className="min-h-0 flex-1 overflow-y-auto px-4 sm:px-6">
        {loadError ? (
          <div className="text-destructive bg-destructive/10 mt-4 rounded-lg px-4 py-3 text-sm">
            {loadErrorMessage || t("chat.historyLoadFailed")}
          </div>
        ) : sessions.length === 0 ? (
          <div className="text-muted-foreground flex flex-col items-center gap-3 py-16 text-sm">
            <IconMessageCircle className="size-10 opacity-30" />
            <p>{t("chat.noHistory")}</p>
          </div>
        ) : (
          <div className="divide-border/50 divide-y py-2">
            {sessions.map((session) => {
              const isActive = session.id === activeSessionId
              return (
                <div
                  key={session.id}
                  className={`group flex cursor-pointer items-center justify-between rounded-lg px-3 py-3 transition-colors hover:bg-muted/50 ${isActive ? "bg-accent/60" : ""}`}
                  onClick={() => void handleSwitchSession(session.id)}
                >
                  <div className="min-w-0 flex-1">
                    <p className={`truncate text-sm font-medium ${isActive ? "text-foreground" : "text-foreground/90"}`}>
                      {session.title || session.preview || t("chat.history")}
                    </p>
                    <p className="text-muted-foreground mt-0.5 text-xs">
                      {t("chat.messagesCount", { count: session.message_count })}
                      {" · "}
                      {dayjs(session.updated).fromNow()}
                    </p>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="text-muted-foreground hover:text-destructive ml-2 size-7 shrink-0 opacity-0 transition-opacity group-hover:opacity-100"
                    onClick={(e) => {
                      e.stopPropagation()
                      void handleDeleteSession(session.id)
                    }}
                    title={t("chat.deleteSession")}
                  >
                    <IconTrash className="size-3.5" />
                  </Button>
                </div>
              )
            })}
            {hasMore && (
              <div
                ref={observerRef}
                className="text-muted-foreground py-3 text-center text-xs"
              >
                <IconLoader2 className="mx-auto size-4 animate-spin" />
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
