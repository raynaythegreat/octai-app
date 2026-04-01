import {
  IconBookmark,
  IconBolt,
  IconLink,
  IconLoader2,
  IconPlug,
  IconSparkles,
  IconTools,
} from "@tabler/icons-react"
import * as React from "react"
import { Link } from "@tanstack/react-router"

import { getReferenceURLs } from "@/api/reference-urls"
import { PageHeader } from "@/components/page-header"
import { Badge } from "@/components/ui/badge"

interface Skill {
  name: string
  description?: string
  type?: string
}

interface Tool {
  name: string
  description: string
  category: string
  status: string
}

interface MCPServer {
  name: string
  enabled: boolean
  type?: string
}

interface CapabilityCardProps {
  icon: React.ComponentType<{ className?: string }>
  title: string
  count: number
  href: string
  color: string
  items: { label: string; sub?: string }[]
  loading: boolean
}

function CapabilityCard({
  icon: Icon,
  title,
  count,
  href,
  color,
  items,
  loading,
}: CapabilityCardProps) {
  return (
    <Link
      to={href as "/agent/skills"}
      className="bg-card border-border/60 hover:bg-muted/30 flex flex-col gap-3 rounded-xl border p-4 transition-colors"
    >
      <div className="flex items-center justify-between">
        <div className={`flex items-center gap-2 ${color}`}>
          <Icon className="size-5" />
          <span className="font-medium">{title}</span>
        </div>
        {loading ? (
          <IconLoader2 className="text-muted-foreground size-4 animate-spin" />
        ) : (
          <Badge variant="secondary" className="tabular-nums">
            {count}
          </Badge>
        )}
      </div>
      {!loading && items.length > 0 && (
        <ul className="flex flex-col gap-1">
          {items.slice(0, 5).map((item, i) => (
            <li key={i} className="flex items-baseline gap-2">
              <span className="bg-muted-foreground/20 mt-1.5 size-1.5 shrink-0 rounded-full" />
              <span className="text-foreground/80 truncate text-xs">
                {item.label}
                {item.sub && (
                  <span className="text-muted-foreground ml-1">{item.sub}</span>
                )}
              </span>
            </li>
          ))}
          {items.length > 5 && (
            <li className="text-muted-foreground pl-3.5 text-xs">
              +{items.length - 5} more
            </li>
          )}
        </ul>
      )}
      {!loading && items.length === 0 && (
        <p className="text-muted-foreground text-xs">None configured yet.</p>
      )}
    </Link>
  )
}

export function CapabilitiesPage() {
  const [skills, setSkills] = React.useState<Skill[]>([])
  const [tools, setTools] = React.useState<Tool[]>([])
  const [mcpServers, setMcpServers] = React.useState<MCPServer[]>([])
  const [refCount, setRefCount] = React.useState(0)
  const [loading, setLoading] = React.useState(true)

  React.useEffect(() => {
    const fetchAll = async () => {
      try {
        await Promise.all([
          fetch("/api/skills")
            .then((r) => (r.ok ? r.json() : null))
            .then((d) => d?.skills && setSkills(d.skills)),
          fetch("/api/tools")
            .then((r) => (r.ok ? r.json() : null))
            .then((d) => d?.tools && setTools(d.tools.filter((t: Tool) => t.status === "enabled"))),
          fetch("/api/config")
            .then((r) => (r.ok ? r.json() : null))
            .then((d) => {
              const servers = d?.tools?.mcp?.servers as Record<string, MCPServer> | undefined
              if (servers) {
                setMcpServers(
                  Object.entries(servers).map(([name, cfg]) => ({
                    name,
                    enabled: cfg.enabled !== false,
                    type: cfg.type,
                  })),
                )
              }
            }),
          getReferenceURLs()
            .then((d) => setRefCount(d.references.length))
            .catch(() => {}),
        ])
      } finally {
        setLoading(false)
      }
    }
    void fetchAll()
  }, [])

  const enabledTools = tools.filter((t) => t.status === "enabled")
  const enabledMcp = mcpServers.filter((s) => s.enabled)

  return (
    <div className="flex h-full flex-col">
      <PageHeader title="Agent Capabilities" />
      <div className="flex min-h-0 flex-1 flex-col gap-4 p-4 md:p-6 overflow-y-auto">
        <p className="text-muted-foreground shrink-0 text-sm">
          Overview of everything your agent can do — skills, tools, MCP servers, and reference links.
        </p>

        {/* Summary row */}
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 shrink-0">
          {[
            { label: "Skills", value: skills.length, icon: IconSparkles, color: "text-violet-400" },
            { label: "Tools", value: enabledTools.length, icon: IconTools, color: "text-blue-400" },
            { label: "MCP Servers", value: enabledMcp.length, icon: IconPlug, color: "text-green-400" },
            { label: "References", value: refCount, icon: IconBookmark, color: "text-orange-400" },
          ].map(({ label, value, icon: Icon, color }) => (
            <div
              key={label}
              className="bg-card border-border/60 flex flex-col items-center gap-1 rounded-xl border p-4"
            >
              <Icon className={`size-6 ${color}`} />
              {loading ? (
                <IconLoader2 className="text-muted-foreground size-4 animate-spin" />
              ) : (
                <span className="text-2xl font-bold tabular-nums">{value}</span>
              )}
              <span className="text-muted-foreground text-xs">{label}</span>
            </div>
          ))}
        </div>

        {/* Detail cards */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <CapabilityCard
            icon={IconSparkles}
            title="Skills"
            count={skills.length}
            href="/agent/skills"
            color="text-violet-400"
            loading={loading}
            items={skills.map((s) => ({
              label: s.name,
              sub: s.description ? `— ${s.description.slice(0, 50)}` : undefined,
            }))}
          />
          <CapabilityCard
            icon={IconTools}
            title="Enabled Tools"
            count={enabledTools.length}
            href="/agent/tools"
            color="text-blue-400"
            loading={loading}
            items={enabledTools.map((t) => ({
              label: t.name,
              sub: `(${t.category})`,
            }))}
          />
          <CapabilityCard
            icon={IconPlug}
            title="MCP Servers"
            count={enabledMcp.length}
            href="/mcp"
            color="text-green-400"
            loading={loading}
            items={enabledMcp.map((s) => ({
              label: s.name,
              sub: s.type ? `(${s.type})` : undefined,
            }))}
          />
          <CapabilityCard
            icon={IconBookmark}
            title="Reference URLs"
            count={refCount}
            href="/agent/reference-url"
            color="text-orange-400"
            loading={loading}
            items={[]}
          />
        </div>

        {/* Slash commands hint */}
        {skills.length > 0 && (
          <div className="bg-muted/30 border-border/40 shrink-0 rounded-xl border p-4">
            <div className="mb-2 flex items-center gap-2">
              <IconBolt className="text-yellow-400 size-4" />
              <span className="text-sm font-medium">Slash Commands</span>
            </div>
            <p className="text-muted-foreground mb-3 text-xs">
              Use any skill directly in chat by typing <code className="bg-muted rounded px-1">/use &lt;skill&gt;</code> or just <code className="bg-muted rounded px-1">/&lt;skill&gt;</code> and selecting from the popup.
            </p>
            <div className="flex flex-wrap gap-1.5">
              {skills.slice(0, 12).map((s) => (
                <span
                  key={s.name}
                  className="bg-muted text-muted-foreground rounded px-2 py-0.5 font-mono text-[11px]"
                >
                  /{s.name}
                </span>
              ))}
              {skills.length > 12 && (
                <span className="text-muted-foreground text-[11px]">
                  +{skills.length - 12} more
                </span>
              )}
            </div>
          </div>
        )}

        {/* AI URL link */}
        <div className="bg-muted/20 border-border/40 shrink-0 rounded-xl border p-4">
          <div className="mb-1 flex items-center gap-2">
            <IconLink className="text-cyan-400 size-4" />
            <span className="text-sm font-medium">Add More Capabilities</span>
          </div>
          <p className="text-muted-foreground text-xs">
            Use the{" "}
            <Link to="/agent/ai-url" className="text-cyan-400 hover:underline">
              AI URL Scanner
            </Link>{" "}
            to discover and integrate skills, tools, and MCP servers from any GitHub repo or website.
          </p>
        </div>
      </div>
    </div>
  )
}
