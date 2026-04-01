import { IconArrowUp, IconBrain, IconGlobe, IconHammer, IconMessageCircle, IconMicrophone, IconPaperclip, IconPhoto } from "@tabler/icons-react"
import type { KeyboardEvent } from "react"
import * as React from "react"
import { useTranslation } from "react-i18next"
import TextareaAutosize from "react-textarea-autosize"

import { Button } from "@/components/ui/button"
import { ImageGenDialog } from "@/components/chat/image-gen-dialog"
import { cn } from "@/lib/utils"

interface ChatComposerProps {
  input: string
  onInputChange: (value: string) => void
  onSend: () => void
  isConnected: boolean
  hasDefaultModel: boolean
  onSendWithAttachments?: (content: string, attachments: { file: File; dataUrl?: string }[]) => void
  chatMode?: "chat" | "plan" | "build"
  onCycleMode?: () => void
  webSearch?: boolean
  onToggleWebSearch?: () => void
}

// Slash commands supported by the gateway
interface SlashCommand {
  name: string
  usage: string
  description: string
  aliases?: string[]
}

const SLASH_COMMANDS: SlashCommand[] = [
  { name: "help",      usage: "/help",                                     description: "Show all available commands" },
  { name: "clear",     usage: "/clear",                                    description: "Clear the chat history" },
  { name: "fast",      usage: "/fast",                                     description: "Toggle fast mode" },
  { name: "model",     usage: "/model [name]",                             description: "Show or switch the current model" },
  { name: "switch",    usage: "/switch model to <name>",                   description: "Switch to a different model" },
  { name: "think",     usage: "/think [off|low|medium|high|xhigh|adaptive]", description: "Set thinking / reasoning level" },
  { name: "use",       usage: "/use <skill> [message]",                    description: "Run an installed skill" },
  { name: "list",      usage: "/list [models|channels|agents|skills]",     description: "List available options" },
  { name: "show",      usage: "/show [model|channel|agents]",              description: "Show current configuration" },
  { name: "status",    usage: "/status",                                   description: "Show current agent settings" },
  { name: "memory",    usage: "/memory <query>",                           description: "Search agent memory" },
  { name: "reload",    usage: "/reload",                                   description: "Reload the configuration file" },
  { name: "check",     usage: "/check channel <name>",                     description: "Check channel availability" },
  { name: "start",     usage: "/start",                                    description: "Start the bot" },
]

// Skills are fetched dynamically from the API at runtime

// Minimal Web Speech API types (not in all TS DOM lib versions)
type SpeechRecognitionCtor = new () => {
  continuous: boolean
  interimResults: boolean
  onstart: (() => void) | null
  onend: (() => void) | null
  onerror: (() => void) | null
  onresult:
    | ((ev: {
        resultIndex: number
        results: { length: number; [i: number]: { [0]: { transcript: string } } }
      }) => void)
    | null
  start(): void
  stop(): void
}

declare global {
  interface Window {
    SpeechRecognition?: SpeechRecognitionCtor
    webkitSpeechRecognition?: SpeechRecognitionCtor
  }
}

export function ChatComposer({
  input,
  onInputChange,
  onSend,
  isConnected,
  hasDefaultModel,
  onSendWithAttachments,
  chatMode = "chat",
  onCycleMode,
  webSearch = false,
  onToggleWebSearch,
}: ChatComposerProps) {
  const { t } = useTranslation()
  const canInput = isConnected && hasDefaultModel
  const recognitionRef = React.useRef<InstanceType<SpeechRecognitionCtor> | null>(null)
  const [isListening, setIsListening] = React.useState(false)
  const [speechSupported] = React.useState(
    () =>
      typeof window !== "undefined" &&
      !!(window.SpeechRecognition || window.webkitSpeechRecognition),
  )

  // Attachment state
  const [attachments, setAttachments] = React.useState<{ file: File; dataUrl?: string }[]>([])
  const fileInputRef = React.useRef<HTMLInputElement>(null)

  // Image generation dialog state
  const [showImageGen, setShowImageGen] = React.useState(false)

  // Dynamic skill commands loaded from API
  const [skillCommands, setSkillCommands] = React.useState<SlashCommand[]>([])

  const fetchSkills = React.useCallback(() => {
    fetch("/api/skills")
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (!data?.skills) return
        setSkillCommands(
          (data.skills as { name: string; description?: string }[]).map((s) => ({
            name: s.name,
            usage: `/use ${s.name}`,
            description: s.description || `Run ${s.name} skill`,
            aliases: [s.name],
          })),
        )
      })
      .catch(() => {})
  }, [])

  React.useEffect(() => {
    fetchSkills()
    window.addEventListener("skills-updated", fetchSkills)
    return () => window.removeEventListener("skills-updated", fetchSkills)
  }, [fetchSkills])

  const allCommands = React.useMemo(
    () => [...SLASH_COMMANDS, ...skillCommands],
    [skillCommands],
  )

  const hasExactCommandMatch = React.useMemo(() => {
    if (!input.startsWith("/")) return false

    const normalized = input.trim().slice(1).toLowerCase()
    if (!normalized) return false

    return allCommands.some((cmd) => {
      const candidates = [
        cmd.name.toLowerCase(),
        cmd.usage.replace(/^\//, "").toLowerCase(),
        ...(cmd.aliases ?? []).map((alias) => alias.toLowerCase()),
      ]

      return candidates.some((candidate) => {
        if (!candidate) return false
        return normalized === candidate || normalized.startsWith(`${candidate} `)
      })
    })
  }, [allCommands, input])

  // Slash command autocomplete state
  const [selectedIndex, setSelectedIndex] = React.useState(0)
  const listRef = React.useRef<HTMLUListElement>(null)

  const suggestions = React.useMemo<SlashCommand[]>(() => {
    if (!input.startsWith("/")) return []
    const query = input.slice(1).toLowerCase()
    if (query === "") return allCommands
    return allCommands
      .map((cmd) => {
        const matchTerms = [
          cmd.name.toLowerCase(),
          cmd.usage.replace(/^\//, "").toLowerCase(),
          ...(cmd.aliases ?? []).map((alias) => alias.toLowerCase()),
        ]

        const score = matchTerms.reduce((best, term) => {
          if (term === query) return 100
          if (term.startsWith(query)) return Math.max(best, 60)
          if (term.includes(query)) return Math.max(best, 30)

          let qi = 0
          for (let ci = 0; ci < term.length && qi < query.length; ci++) {
            if (term[ci] === query[qi]) qi++
          }
          if (qi === query.length) return Math.max(best, 10)

          return best
        }, 0)

        return { cmd, score }
      })
      .filter((x) => x.score > 0)
      .sort((a, b) => b.score - a.score)
      .map((x) => x.cmd)
  }, [input, allCommands])

  const showPopup = suggestions.length > 0

  // Reset selection when suggestions change
  React.useEffect(() => {
    setSelectedIndex(0)
  }, [suggestions.length])

  // Scroll active item into view
  React.useEffect(() => {
    if (!listRef.current) return
    const item = listRef.current.children[selectedIndex] as HTMLElement | undefined
    item?.scrollIntoView({ block: "nearest" })
  }, [selectedIndex])

  const applyCommand = (cmd: SlashCommand) => {
    // Fill in the command usage. For commands with required args, put the
    // cursor after the command name so the user can type the argument.
    const hasArgs = cmd.usage.includes("<") || cmd.usage.includes("[")
    onInputChange(hasArgs ? `/${cmd.name} ` : cmd.usage)
  }

  const handleSend = () => {
    if (!input.trim() && attachments.length === 0) return
    if (attachments.length > 0 && onSendWithAttachments) {
      onSendWithAttachments(input.trim(), attachments)
      setAttachments([])
      onInputChange("")
    } else {
      onSend()
    }
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.nativeEvent.isComposing) return

    if (showPopup) {
      if (e.key === "ArrowDown") {
        e.preventDefault()
        setSelectedIndex((i) => Math.min(i + 1, suggestions.length - 1))
        return
      }
      if (e.key === "ArrowUp") {
        e.preventDefault()
        setSelectedIndex((i) => Math.max(i - 1, 0))
        return
      }
      if (e.key === "Tab") {
        e.preventDefault()
        applyCommand(suggestions[selectedIndex])
        return
      }
      if (e.key === "Enter" && !e.shiftKey && !hasExactCommandMatch) {
        e.preventDefault()
        applyCommand(suggestions[selectedIndex])
        return
      }
      if (e.key === "Escape") {
        e.preventDefault()
        onInputChange("")
        return
      }
    }

    // Tab with empty input → cycle Chat/Plan/Build mode
    if (e.key === "Tab" && !input.trim()) {
      e.preventDefault()
      onCycleMode?.()
      return
    }

    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const toggleListening = () => {
    if (isListening) {
      recognitionRef.current?.stop()
      return
    }
    const SR = window.SpeechRecognition ?? window.webkitSpeechRecognition
    if (!SR) return
    const rec = new SR()
    rec.continuous = false
    rec.interimResults = true
    rec.onstart = () => setIsListening(true)
    rec.onend = () => setIsListening(false)
    rec.onerror = () => setIsListening(false)
    rec.onresult = (ev) => {
      let transcript = ""
      for (let i = ev.resultIndex; i < ev.results.length; i++) {
        transcript += ev.results[i][0].transcript
      }
      onInputChange(transcript)
    }
    recognitionRef.current = rec
    rec.start()
  }

  return (
    <div className="bg-background shrink-0 px-4 pt-3 pb-[calc(1rem+env(safe-area-inset-bottom))] md:px-6 md:pb-6 lg:px-10 xl:px-16">
      <div className="mx-auto max-w-[1040px]">
        {/* Slash command popup */}
        {showPopup && (
          <div className="bg-card border-border/80 mb-1 overflow-hidden rounded-xl border shadow-lg">
            <ul ref={listRef} className="max-h-64 overflow-y-auto py-1" role="listbox">
              {suggestions.map((cmd, i) => (
                <li
                  key={cmd.name}
                  role="option"
                  aria-selected={i === selectedIndex}
                  className={cn(
                    "flex cursor-pointer items-baseline gap-3 px-4 py-2 text-sm transition-colors",
                    i === selectedIndex
                      ? "bg-violet-500/10 text-foreground"
                      : "hover:bg-muted/60 text-foreground",
                  )}
                  onMouseDown={(e) => {
                    // Use mousedown so blur doesn't fire before click
                    e.preventDefault()
                    applyCommand(cmd)
                  }}
                  onMouseEnter={() => setSelectedIndex(i)}
                >
                  <span className="font-mono text-violet-400 shrink-0">{cmd.usage}</span>
                  <span className="text-muted-foreground truncate">{cmd.description}</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Composer card */}
        <div className="bg-card border-border/80 flex flex-col rounded-2xl border p-3 shadow-md">
          {/* Hidden file input */}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*,.pdf,.txt,.md,.json,.csv,.py,.js,.ts"
            multiple
            className="hidden"
            onChange={(e) => {
              const files = Array.from(e.target.files ?? [])
              files.forEach(file => {
                if (file.type.startsWith("image/")) {
                  const reader = new FileReader()
                  reader.onload = (ev) => {
                    setAttachments(prev => [...prev, { file, dataUrl: ev.target?.result as string }])
                  }
                  reader.readAsDataURL(file)
                } else {
                  setAttachments(prev => [...prev, { file }])
                }
              })
              e.target.value = ""
            }}
          />

          <TextareaAutosize
            value={input}
            onChange={(e) => onInputChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t("chat.placeholder")}
            disabled={!canInput}
            className={cn(
              "placeholder:text-muted-foreground max-h-[200px] min-h-[60px] resize-none border-0 bg-transparent px-2 py-1 text-[15px] shadow-none transition-colors focus-visible:ring-0 focus-visible:outline-none dark:bg-transparent",
              !canInput && "cursor-not-allowed",
            )}
            minRows={1}
            maxRows={8}
          />

          {/* Attachment previews */}
          {attachments.length > 0 && (
            <div className="flex flex-wrap gap-2 px-2 pt-2">
              {attachments.map((att, i) => (
                <div key={i} className="relative flex items-center gap-1 rounded-lg bg-muted/60 px-2 py-1 text-xs">
                  {att.dataUrl ? (
                    <img src={att.dataUrl} className="size-8 rounded object-cover" alt={att.file.name} />
                  ) : (
                    <span className="max-w-[120px] truncate">{att.file.name}</span>
                  )}
                  <button
                    type="button"
                    className="ml-1 text-muted-foreground hover:text-foreground"
                    onClick={() => setAttachments(prev => prev.filter((_, j) => j !== i))}
                  >
                    ×
                  </button>
                </div>
              ))}
            </div>
          )}

          <div className="mt-2 flex items-center justify-between px-1">
            {/* Left: Chat/Plan/Build mode cycle + Web search toggle */}
            <div className="flex items-center gap-2">
              <Button
                type="button"
                size="sm"
                variant="ghost"
                className={cn(
                  "h-7 gap-1.5 rounded-full px-2.5 text-xs font-medium transition-all",
                  chatMode === "plan" && "bg-amber-500/15 text-amber-400 hover:bg-amber-500/20",
                  chatMode === "chat" && "bg-violet-500/15 text-violet-400 hover:bg-violet-500/20",
                  chatMode === "build" && "bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20",
                )}
                onClick={onCycleMode}
                disabled={!canInput}
                title={`${chatMode === "build" ? "Build" : chatMode === "plan" ? "Plan" : "Chat"} mode (Tab to cycle)`}
              >
                {chatMode === "plan" ? (
                  <IconBrain className="size-3.5" />
                ) : chatMode === "chat" ? (
                  <IconMessageCircle className="size-3.5" />
                ) : (
                  <IconHammer className="size-3.5" />
                )}
                {chatMode === "plan" ? "Plan" : chatMode === "chat" ? "Chat" : "Build"}
              </Button>

              {/* Web search toggle */}
              <Button
                type="button"
                size="sm"
                variant="ghost"
                className={cn(
                  "h-7 w-7 rounded-full p-0 transition-all",
                  webSearch
                    ? "bg-sky-500/15 text-sky-400 hover:bg-sky-500/20 border border-sky-400/50"
                    : "text-muted-foreground hover:text-sky-400 hover:bg-sky-500/10",
                )}
                onClick={onToggleWebSearch}
                disabled={!canInput}
                title={webSearch ? "Web search enabled" : "Enable web search"}
              >
                <IconGlobe className="size-3.5" />
              </Button>
            </div>

            {/* Right: media buttons + send */}
            <div className="flex items-center gap-1">
              {speechSupported && (
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  className={cn(
                    "size-8 rounded-full transition-all",
                    isListening
                      ? "bg-violet-500/20 text-violet-400 animate-pulse"
                      : "text-muted-foreground hover:text-violet-400 hover:bg-violet-500/10",
                  )}
                  onClick={toggleListening}
                  disabled={!canInput}
                  title={isListening ? t("chat.voice.stopRecording") : t("chat.voice.voiceInput")}
                >
                  <IconMicrophone className="size-4" />
                </Button>
              )}
              <Button
                type="button"
                size="icon"
                variant="ghost"
                className="size-8 rounded-full text-muted-foreground hover:text-violet-400 hover:bg-violet-500/10"
                onClick={() => fileInputRef.current?.click()}
                disabled={!canInput}
                title="Attach files"
              >
                <IconPaperclip className="size-4" />
              </Button>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                className="size-8 rounded-full text-muted-foreground hover:text-violet-400 hover:bg-violet-500/10"
                onClick={() => setShowImageGen(true)}
                disabled={!canInput}
                title="Generate image"
              >
                <IconPhoto className="size-4" />
              </Button>
              <Button
                size="icon"
                className="size-8 rounded-full bg-violet-500 text-white transition-transform hover:bg-violet-600 active:scale-95"
                onClick={handleSend}
                disabled={(!input.trim() && attachments.length === 0) || !canInput}
              >
                <IconArrowUp className="size-4" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <ImageGenDialog
        open={showImageGen}
        onClose={() => setShowImageGen(false)}
        onInsert={(dataUrl, _prompt) => {
          setAttachments(prev => [...prev, { file: new File([dataUrl], "generated.png", { type: "image/png" }), dataUrl }])
          setShowImageGen(false)
        }}
      />
    </div>
  )
}
