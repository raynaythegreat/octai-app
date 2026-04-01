import { IconPlus } from "@tabler/icons-react"
import { useAtomValue } from "jotai"
import { useCallback, useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import { getSessionHistory } from "@/api/sessions"
import { AssistantMessage } from "@/components/chat/assistant-message"
import { ChannelSelector } from "@/components/chat/channel-selector"
import { ChatComposer } from "@/components/chat/chat-composer"
import { ChatEmptyState } from "@/components/chat/chat-empty-state"
import { ModelSelector } from "@/components/chat/model-selector"
import { SessionHistoryMenu } from "@/components/chat/session-history-menu"
import { ThinkingLevelSelector } from "@/components/chat/thinking-level-selector"
import { TypingIndicator } from "@/components/chat/typing-indicator"
import { UserMessage } from "@/components/chat/user-message"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { useChatModels } from "@/hooks/use-chat-models"
import { useGateway } from "@/hooks/use-gateway"
import { usePicoChat } from "@/hooks/use-pico-chat"
import { useSessionHistory } from "@/hooks/use-session-history"
import { chatAtom, setThinkingLevel, type ThinkingLevel } from "@/store/chat"

export function ChatPage() {
  const { t } = useTranslation()
  const scrollRef = useRef<HTMLDivElement>(null)
  const [isAtBottom, setIsAtBottom] = useState(true)
  const [hasScrolled, setHasScrolled] = useState(false)
  const [input, setInput] = useState("")
  const [activeChannel, setActiveChannel] = useState("pico")
  const [viewingSessionId, setViewingSessionId] = useState<string | null>(null)
  const [viewingMessages, setViewingMessages] = useState<{ role: string; content: string }[] | null>(null)
  const [chatMode, setChatMode] = useState<"chat" | "plan" | "build">("chat")
  const [webSearch, setWebSearch] = useState(false)

  const readOnly = activeChannel !== "pico"

  const {
    messages,
    connectionState,
    isTyping,
    activeSessionId,
    sendMessage,
    switchSession,
    newChat,
  } = usePicoChat()

  const { state: gwState } = useGateway()
  const isGatewayRunning = gwState === "running"
  const isChatConnected = connectionState === "connected"

  const {
    defaultModelName,
    hasSavedModels,
    hasChatEnabledModels,
    hasAvailableModels,
    apiKeyModels,
    oauthModels,
    localModels,
    handleSetDefault,
    isAutoMode,
    toggleAutoMode,
  } = useChatModels({ isConnected: isGatewayRunning })
  const canSend = isChatConnected && (isAutoMode || Boolean(defaultModelName))

  const { thinkingLevel } = useAtomValue(chatAtom)
  const allModels = [...apiKeyModels, ...oauthModels, ...localModels]

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
    channel: activeChannel !== "pico" ? activeChannel : undefined,
  })

  const loadReadOnlySession = useCallback(
    async (id: string) => {
      try {
        const detail = await getSessionHistory(id, activeChannel)
        setViewingSessionId(id)
        setViewingMessages(detail.messages)
      } catch (err) {
        console.error("Failed to load session:", err)
      }
    },
    [activeChannel],
  )

  const syncScrollState = (element: HTMLDivElement) => {
    const { scrollTop, scrollHeight, clientHeight } = element
    setHasScrolled(scrollTop > 0)
    setIsAtBottom(scrollHeight - scrollTop <= clientHeight + 10)
  }

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    syncScrollState(e.currentTarget)
  }

  useEffect(() => {
    if (scrollRef.current) {
      if (isAtBottom) {
        scrollRef.current.scrollTop = scrollRef.current.scrollHeight
      }
      syncScrollState(scrollRef.current)
    }
  }, [messages, isTyping, isAtBottom])

  const applyModePrefix = (content: string) => {
    if (chatMode === "plan") {
      return `Before taking any action, write a clear numbered plan of what you will do. Then execute it step by step.\n\n${content}`
    }
    if (chatMode === "build") {
      return `Focus on executing tasks efficiently. Write code, run commands, and make changes directly.\n\n${content}`
    }
    return content
  }

  const cycleChatMode = () => {
    setChatMode((m) => (m === "chat" ? "plan" : m === "plan" ? "build" : "chat"))
  }

  const handleSend = () => {
    if (!input.trim() || !canSend) return
    if (sendMessage(applyModePrefix(input.trim()), { webSearch })) {
      setInput("")
    }
  }

  const handleSendWithAttachments = async (
    content: string,
    attachments: { file: File; dataUrl?: string }[],
  ) => {
    if (!canSend) return
    let fullContent = content
    for (const att of attachments) {
      if (att.dataUrl && att.dataUrl.startsWith("data:image/")) {
        // Note the attached image by name; vision-capable models receive context via the UI
        fullContent += `\n\n[Attached image: ${att.file.name}]`
      } else {
        // For text files, inline the content
        try {
          const text = await att.file.text()
          fullContent += `\n\n\`\`\`\n${att.file.name}:\n${text}\`\`\``
        } catch {
          fullContent += `\n\n[Attached file: ${att.file.name}]`
        }
      }
    }
    if (sendMessage(applyModePrefix(fullContent), { webSearch })) {
      setInput("")
    }
  }

  return (
    <div className="bg-background/95 flex h-full flex-col">
      <PageHeader
        title={t("navigation.chat")}
        className={`transition-shadow ${
          hasScrolled ? "shadow-sm" : "shadow-none"
        }`}
        titleExtra={
          <div className="flex items-center gap-2">
            {hasAvailableModels && (
              <>
                <ModelSelector
                  defaultModelName={defaultModelName}
                  apiKeyModels={apiKeyModels}
                  oauthModels={oauthModels}
                  localModels={localModels}
                  onValueChange={handleSetDefault}
                  isAutoMode={isAutoMode}
                  toggleAutoMode={toggleAutoMode}
                />
                <ThinkingLevelSelector
                  models={allModels}
                  defaultModelName={defaultModelName}
                  isAutoMode={isAutoMode}
                  thinkingLevel={thinkingLevel}
                  onThinkingLevelChange={(level: ThinkingLevel) => setThinkingLevel(level)}
                />
              </>
            )}
            <ChannelSelector
              activeChannel={activeChannel}
              onChannelChange={(ch) => {
                setActiveChannel(ch)
                setViewingSessionId(null)
                setViewingMessages(null)
              }}
            />
          </div>
        }
      >
        <Button
          variant="secondary"
          size="sm"
          onClick={
            readOnly
              ? () => {
                  setViewingSessionId(null)
                  setViewingMessages(null)
                }
              : newChat
          }
          className="h-9 gap-2"
        >
          <IconPlus className="size-4" />
          <span className="hidden sm:inline">{t("chat.newChat")}</span>
        </Button>

        <SessionHistoryMenu
          sessions={sessions}
          activeSessionId={readOnly ? (viewingSessionId ?? "") : activeSessionId}
          hasMore={hasMore}
          loadError={loadError}
          loadErrorMessage={loadErrorMessage}
          observerRef={observerRef}
          onOpenChange={(open) => {
            if (open) {
              void loadSessions(true)
            }
          }}
          onSwitchSession={readOnly ? loadReadOnlySession : switchSession}
          onDeleteSession={handleDeleteSession}
        />
      </PageHeader>

      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="min-h-0 flex-1 overflow-y-auto px-4 py-6 md:px-6 lg:px-10 xl:px-16"
      >
        <div className={`mx-auto flex w-full max-w-[1040px] flex-col gap-5 pb-6 ${!readOnly && messages.length === 0 && !isTyping ? 'h-full' : ''}`}>
          {!readOnly && messages.length === 0 && !isTyping && (
            <ChatEmptyState
              hasSavedModels={hasSavedModels}
              hasChatEnabledModels={hasChatEnabledModels}
              hasAvailableModels={hasAvailableModels}
              defaultModelName={defaultModelName}
              isAutoMode={isAutoMode}
              isConnected={isGatewayRunning}
            />
          )}

          {!readOnly &&
            messages.map((msg) => (
              <div key={msg.id} className="flex w-full justify-center">
                {msg.role === "assistant" ? (
                  <AssistantMessage
                    content={msg.content}
                    timestamp={msg.timestamp}
                    meta={msg.meta}
                  />
                ) : (
                  <UserMessage content={msg.content} />
                )}
              </div>
            ))}

          {!readOnly && isTyping && <TypingIndicator />}

          {readOnly && !viewingMessages && (
            <div className="flex h-full items-center justify-center text-muted-foreground">
              <p>{t("chat.channel.noHistory", { channel: activeChannel.toUpperCase() })}</p>
            </div>
          )}

          {readOnly &&
            viewingMessages &&
            viewingMessages.map((msg, i) => (
              <div key={i} className="flex w-full justify-center">
                {msg.role === "assistant" ? (
                  <AssistantMessage content={msg.content} />
                ) : (
                  <UserMessage content={msg.content} />
                )}
              </div>
            ))}
        </div>
      </div>

      {readOnly ? (
        <div className="border-t px-4 py-3 text-center text-sm text-muted-foreground">
          {t("chat.channel.readOnly")}
        </div>
      ) : (
        <ChatComposer
          input={input}
          onInputChange={setInput}
          onSend={handleSend}
          isConnected={isChatConnected}
          hasDefaultModel={hasAvailableModels && (isAutoMode || Boolean(defaultModelName))}
          onSendWithAttachments={(content, attachments) => {
            void handleSendWithAttachments(content, attachments)
          }}
          chatMode={chatMode}
          onCycleMode={cycleChatMode}
          webSearch={webSearch}
          onToggleWebSearch={() => setWebSearch((v) => !v)}
        />
      )}
    </div>
  )
}
