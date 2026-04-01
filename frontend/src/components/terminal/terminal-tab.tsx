import { FitAddon } from "@xterm/addon-fit"
import { SearchAddon } from "@xterm/addon-search"
import { WebLinksAddon } from "@xterm/addon-web-links"
import { Terminal } from "@xterm/xterm"
import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import { PageHeader } from "@/components/page-header"

import "@xterm/xterm/css/xterm.css"
import "./terminal.css"

const TERMINAL_WS_URL = `ws://${window.location.hostname}:18790/ws/terminal`
const RECONNECT_DELAY_MS = 2000
const MAX_RECONNECT_DELAY_MS = 30000

type ConnectionStatus = "connected" | "disconnected" | "connecting"

const OCTAI_THEME = {
  background: "#1a1a2e",
  foreground: "#e0e0e0",
  cursor: "#22c55e",
  cursorAccent: "#1a1a2e",
  selectionBackground: "rgba(34, 197, 94, 0.3)",
  selectionForeground: "#ffffff",
  black: "#1a1a2e",
  red: "#ef4444",
  green: "#22c55e",
  yellow: "#eab308",
  blue: "#3b82f6",
  magenta: "#a855f7",
  cyan: "#06b6d4",
  white: "#e0e0e0",
  brightBlack: "#6b7280",
  brightRed: "#f87171",
  brightGreen: "#4ade80",
  brightYellow: "#facc15",
  brightBlue: "#60a5fa",
  brightMagenta: "#c084fc",
  brightCyan: "#22d3ee",
  brightWhite: "#ffffff",
}

export function TerminalTab() {
  const { t } = useTranslation()
  const termRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<Terminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const reconnectDelayRef = useRef(RECONNECT_DELAY_MS)
  const [status, setStatus] = useState<ConnectionStatus>("disconnected")

  const connect = () => {
    if (
      wsRef.current &&
      (wsRef.current.readyState === WebSocket.OPEN ||
        wsRef.current.readyState === WebSocket.CONNECTING)
    ) {
      return
    }

    setStatus("connecting")

    try {
      const ws = new WebSocket(TERMINAL_WS_URL)

      ws.onopen = () => {
        setStatus("connected")
        reconnectDelayRef.current = RECONNECT_DELAY_MS

        if (xtermRef.current && fitAddonRef.current) {
          fitAddonRef.current.fit()
          const { cols, rows } = xtermRef.current
          ws.send(
            JSON.stringify({
              type: "resize",
              cols,
              rows,
            }),
          )
        }
      }

      ws.onmessage = (event) => {
        if (xtermRef.current) {
          xtermRef.current.write(event.data)
        }
      }

      ws.onclose = () => {
        setStatus("disconnected")
        wsRef.current = null
        scheduleReconnect()
      }

      ws.onerror = () => {
        ws.close()
      }

      wsRef.current = ws
    } catch {
      setStatus("disconnected")
      scheduleReconnect()
    }
  }

  const scheduleReconnect = () => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
    }

    reconnectTimerRef.current = setTimeout(() => {
      connect()
    }, reconnectDelayRef.current)

    reconnectDelayRef.current = Math.min(
      reconnectDelayRef.current * 1.5,
      MAX_RECONNECT_DELAY_MS,
    )
  }

  useEffect(() => {
    if (!termRef.current) return

    const terminal = new Terminal({
      theme: OCTAI_THEME,
      fontFamily: '"JetBrains Mono", "Fira Code", "Cascadia Code", Menlo, Monaco, "Courier New", monospace',
      fontSize: 14,
      lineHeight: 1.3,
      cursorBlink: true,
      cursorStyle: "bar",
      scrollback: 10000,
      allowProposedApi: true,
      allowTransparency: false,
      macOptionIsMeta: true,
    })

    const fitAddon = new FitAddon()
    const webLinksAddon = new WebLinksAddon()
    const searchAddon = new SearchAddon()

    terminal.loadAddon(fitAddon)
    terminal.loadAddon(webLinksAddon)
    terminal.loadAddon(searchAddon)

    terminal.open(termRef.current)

    xtermRef.current = terminal
    fitAddonRef.current = fitAddon

    const resizeObserver = new ResizeObserver(() => {
      fitAddon.fit()
      if (
        wsRef.current &&
        wsRef.current.readyState === WebSocket.OPEN
      ) {
        const { cols, rows } = terminal
        wsRef.current.send(
          JSON.stringify({
            type: "resize",
            cols,
            rows,
          }),
        )
      }
    })
    resizeObserver.observe(termRef.current)

    terminal.onData((data) => {
      if (
        wsRef.current &&
        wsRef.current.readyState === WebSocket.OPEN
      ) {
        wsRef.current.send(
          JSON.stringify({
            type: "input",
            data,
          }),
        )
      }
    })

    terminal.attachCustomKeyEventHandler((event) => {
      if (event.ctrlKey && event.shiftKey && event.key === "C") {
        const selection = terminal.getSelection()
        if (selection) {
          navigator.clipboard.writeText(selection)
        }
        return false
      }
      if (event.ctrlKey && event.shiftKey && event.key === "V") {
        navigator.clipboard.readText().then((text) => {
          if (
            wsRef.current &&
            wsRef.current.readyState === WebSocket.OPEN
          ) {
            wsRef.current.send(
              JSON.stringify({
                type: "input",
                data: text,
              }),
            )
          }
        })
        return false
      }
      if (event.ctrlKey && event.shiftKey && event.key === "F") {
        return true
      }
      return true
    })

    connect()

    return () => {
      resizeObserver.disconnect()
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
      }
      if (wsRef.current) {
        wsRef.current.onclose = null
        wsRef.current.close()
        wsRef.current = null
      }
      terminal.dispose()
      xtermRef.current = null
      fitAddonRef.current = null
    }
  }, [])

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title={t("navigation.terminal")}
        titleExtra={
          <div className="terminal-status">
            <div
              className={`terminal-status-dot ${status}`}
            />
            <span>
              {status === "connected" &&
                t("terminal.connected")}
              {status === "disconnected" &&
                t("terminal.disconnected")}
              {status === "connecting" &&
                t("terminal.connecting")}
            </span>
          </div>
        }
      />
      <div className="flex flex-1 flex-col overflow-hidden p-4 sm:p-8 pt-0 sm:pt-0">
        <div className="terminal-container flex-1 min-h-0">
          <div ref={termRef} className="h-full w-full" />
        </div>
      </div>
    </div>
  )
}
