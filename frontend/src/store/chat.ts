import { atom, getDefaultStore } from "jotai"

import {
  getInitialActiveSessionId,
  getInitialShowDetailedSteps,
  getInitialThinkingLevel,
  type ThinkingLevel,
  writeStoredSessionId,
  writeStoredShowDetailedSteps,
  writeStoredThinkingLevel,
} from "@/features/chat/state"

export type { ThinkingLevel }

export interface ToolUseBlock {
  tool_name: string
  status: "running" | "done" | "error"
  args_preview?: string
  result_preview?: string
  duration_ms?: number
}

export interface AgentBlock {
  agent_id: string
  agent_name: string
  status: "running" | "done" | "error"
  model?: string
  tokens?: { input: number; output: number }
  tool_uses?: ToolUseBlock[]
}

export interface MessageMeta {
  active_skills?: string[]
  tool_uses?: ToolUseBlock[]
  agents?: AgentBlock[]
}

export interface ChatMessage {
  id: string
  role: "user" | "assistant"
  content: string
  timestamp: number | string
  meta?: MessageMeta
}

export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "error"

export type ThinkingPhase = "thinking" | "references" | "response" | null

export interface ChatStoreState {
  messages: ChatMessage[]
  connectionState: ConnectionState
  isTyping: boolean
  activeSessionId: string
  hasHydratedActiveSession: boolean
  showDetailedSteps: boolean
  thinkingPhase: ThinkingPhase
  thinkingStartTime: number | null
  thinkingReferences: Reference[]
  thinkingLevel: ThinkingLevel
}

export interface Reference {
  id: string
  title: string
  url?: string
  snippet?: string
}

type ChatStorePatch = Partial<ChatStoreState>

const DEFAULT_CHAT_STATE: ChatStoreState = {
  messages: [],
  connectionState: "disconnected",
  isTyping: false,
  activeSessionId: getInitialActiveSessionId(),
  hasHydratedActiveSession: false,
  showDetailedSteps: getInitialShowDetailedSteps(),
  thinkingPhase: null,
  thinkingStartTime: null,
  thinkingReferences: [],
  thinkingLevel: getInitialThinkingLevel(),
}

export const chatAtom = atom<ChatStoreState>(DEFAULT_CHAT_STATE)

const store = getDefaultStore()

export function getChatState() {
  return store.get(chatAtom)
}

export function updateChatStore(
  patch:
    | ChatStorePatch
    | ((prev: ChatStoreState) => ChatStorePatch | ChatStoreState),
) {
  store.set(chatAtom, (prev: ChatStoreState) => {
    const nextPatch = typeof patch === "function" ? patch(prev) : patch
    const next = { ...prev, ...nextPatch }

    if (next.activeSessionId !== prev.activeSessionId) {
      writeStoredSessionId(next.activeSessionId)
    }

    if (next.showDetailedSteps !== prev.showDetailedSteps) {
      writeStoredShowDetailedSteps(next.showDetailedSteps)
    }

    if (next.thinkingLevel !== prev.thinkingLevel) {
      writeStoredThinkingLevel(next.thinkingLevel)
    }

    return next
  })
}

export function getShowDetailedSteps() {
  return store.get(chatAtom).showDetailedSteps
}

export function setShowDetailedSteps(showDetailedSteps: boolean) {
  updateChatStore((prev) => ({ ...prev, showDetailedSteps }))
}

export function startThinkingPhase() {
  updateChatStore((prev) => ({
    ...prev,
    thinkingPhase: "thinking",
    thinkingStartTime: Date.now(),
    thinkingReferences: [],
  }))
}

export function setThinkingReferences(references: Reference[]) {
  updateChatStore((prev) => ({
    ...prev,
    thinkingPhase: "references",
    thinkingReferences: references,
  }))
}

export function showThinkingResponse() {
  updateChatStore((prev) => ({
    ...prev,
    thinkingPhase: "response",
  }))
}

export function clearThinkingPhase() {
  updateChatStore((prev) => ({
    ...prev,
    thinkingPhase: null,
    thinkingStartTime: null,
    thinkingReferences: [],
  }))
}

export function getThinkingPhase() {
  return store.get(chatAtom).thinkingPhase
}

export function getThinkingStartTime() {
  return store.get(chatAtom).thinkingStartTime
}

export function getThinkingReferences() {
  return store.get(chatAtom).thinkingReferences
}

export function getThinkingLevel(): ThinkingLevel {
  return store.get(chatAtom).thinkingLevel
}

export function setThinkingLevel(thinkingLevel: ThinkingLevel) {
  updateChatStore((prev) => ({ ...prev, thinkingLevel }))
}
