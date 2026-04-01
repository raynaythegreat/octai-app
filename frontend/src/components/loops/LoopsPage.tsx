import {
  IconLoader,
  IconPlayerPause,
  IconPlayerPlay,
  IconPlus,
  IconRefresh,
  IconTrash,
} from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { toast } from "sonner"

import { PageHeader } from "@/components/page-header"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Field } from "@/components/shared-form"

// ─── Types ────────────────────────────────────────────────────────────────────

export type LoopStatus = "active" | "paused" | "completed" | "expired"

export interface Loop {
  id: string
  prompt: string
  interval: string
  interval_seconds: number
  run_count: number
  max_runs: number
  last_run?: string
  next_run?: string
  status: LoopStatus
  expires_at?: string
  created_at: string
}

interface CreateLoopPayload {
  prompt: string
  interval_seconds: number
  max_runs: number
  expires_at?: string
}

// ─── Mock data (used when API is unavailable) ─────────────────────────────────

const MOCK_LOOPS: Loop[] = [
  {
    id: "loop-001",
    prompt: "Check the inbox and summarize any new emails from the last hour.",
    interval: "1h",
    interval_seconds: 3600,
    run_count: 12,
    max_runs: 0,
    last_run: new Date(Date.now() - 55 * 60 * 1000).toISOString(),
    next_run: new Date(Date.now() + 5 * 60 * 1000).toISOString(),
    status: "active",
    created_at: new Date(Date.now() - 12 * 3600 * 1000).toISOString(),
  },
  {
    id: "loop-002",
    prompt: "Run a quick system health check and report any anomalies to the ops channel.",
    interval: "5m",
    interval_seconds: 300,
    run_count: 144,
    max_runs: 288,
    last_run: new Date(Date.now() - 4 * 60 * 1000).toISOString(),
    next_run: new Date(Date.now() + 60 * 1000).toISOString(),
    status: "paused",
    created_at: new Date(Date.now() - 24 * 3600 * 1000).toISOString(),
  },
  {
    id: "loop-003",
    prompt: "Fetch latest market data and update the dashboard summary sheet.",
    interval: "30s",
    interval_seconds: 30,
    run_count: 50,
    max_runs: 50,
    last_run: new Date(Date.now() - 35 * 1000).toISOString(),
    next_run: undefined,
    status: "completed",
    created_at: new Date(Date.now() - 2 * 3600 * 1000).toISOString(),
    expires_at: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
  },
]

// ─── API helpers ──────────────────────────────────────────────────────────────

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    let message = `API error: ${res.status} ${res.statusText}`
    try {
      const body = (await res.json()) as { error?: string }
      if (body.error) message = body.error
    } catch {}
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

async function fetchLoops(): Promise<Loop[]> {
  try {
    const res = await apiFetch<{ loops: Loop[] }>("/api/v1/loops")
    return res.loops
  } catch {
    // Fall back to mock data if the endpoint doesn't exist yet
    return MOCK_LOOPS
  }
}

async function createLoop(payload: CreateLoopPayload): Promise<Loop> {
  return apiFetch<Loop>("/api/v1/loops", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  })
}

async function deleteLoop(id: string): Promise<void> {
  await apiFetch<unknown>(`/api/v1/loops/${id}`, { method: "DELETE" })
}

async function pauseLoop(id: string): Promise<Loop> {
  return apiFetch<Loop>(`/api/v1/loops/${id}/pause`, { method: "POST" })
}

async function resumeLoop(id: string): Promise<Loop> {
  return apiFetch<Loop>(`/api/v1/loops/${id}/resume`, { method: "POST" })
}

// ─── Hooks ────────────────────────────────────────────────────────────────────

function useLoops() {
  return useQuery({
    queryKey: ["loops"],
    queryFn: fetchLoops,
    staleTime: 15_000,
    refetchInterval: 30_000,
  })
}

function useCreateLoop() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createLoop,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["loops"] })
      toast.success("Loop created")
    },
    onError: (err: Error) => {
      toast.error(`Failed to create loop: ${err.message}`)
    },
  })
}

function useDeleteLoop() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteLoop,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["loops"] })
      toast.success("Loop deleted")
    },
    onError: (err: Error) => {
      toast.error(`Failed to delete loop: ${err.message}`)
    },
  })
}

function usePauseLoop() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: pauseLoop,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["loops"] })
      toast.success("Loop paused")
    },
    onError: (err: Error) => {
      toast.error(`Failed to pause loop: ${err.message}`)
    },
  })
}

function useResumeLoop() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: resumeLoop,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["loops"] })
      toast.success("Loop resumed")
    },
    onError: (err: Error) => {
      toast.error(`Failed to resume loop: ${err.message}`)
    },
  })
}

// ─── Utilities ────────────────────────────────────────────────────────────────

function truncate(text: string, maxLen: number): string {
  return text.length > maxLen ? `${text.slice(0, maxLen)}…` : text
}

function formatDate(iso?: string): string {
  if (!iso) return "—"
  const d = new Date(iso)
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

function parseIntervalToSeconds(interval: string): number {
  const trimmed = interval.trim().toLowerCase()
  const match = /^(\d+(?:\.\d+)?)\s*(s|m|h|d)?$/.exec(trimmed)
  if (!match) return 0
  const value = parseFloat(match[1])
  const unit = match[2] ?? "s"
  const multipliers: Record<string, number> = { s: 1, m: 60, h: 3600, d: 86400 }
  return Math.round(value * (multipliers[unit] ?? 1))
}

function expiresAtFromOption(option: string): string | undefined {
  if (option === "never") return undefined
  const hoursMap: Record<string, number> = {
    "1h": 1,
    "6h": 6,
    "24h": 24,
    "72h": 72,
  }
  const hours = hoursMap[option]
  if (!hours) return undefined
  return new Date(Date.now() + hours * 3600 * 1000).toISOString()
}

// ─── Status badge ─────────────────────────────────────────────────────────────

const statusStyles: Record<LoopStatus, string> = {
  active:    "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300 border-transparent",
  paused:    "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300 border-transparent",
  completed: "bg-gray-100 text-gray-700 dark:bg-gray-800/40 dark:text-gray-400 border-transparent",
  expired:   "bg-violet-100 text-violet-800 dark:bg-violet-900/30 dark:text-violet-300 border-transparent",
}

function StatusBadge({ status }: { status: LoopStatus }) {
  return (
    <Badge className={statusStyles[status]}>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </Badge>
  )
}

// ─── New Loop form ────────────────────────────────────────────────────────────

interface NewLoopFormState {
  prompt: string
  interval: string
  maxRuns: string
  expiresAfter: string
}

const EMPTY_FORM: NewLoopFormState = {
  prompt: "",
  interval: "1h",
  maxRuns: "",
  expiresAfter: "24h",
}

interface NewLoopSheetProps {
  open: boolean
  onClose: () => void
}

function NewLoopSheet({ open, onClose }: NewLoopSheetProps) {
  const [form, setForm] = useState<NewLoopFormState>(EMPTY_FORM)
  const [errors, setErrors] = useState<Partial<NewLoopFormState>>({})
  const createMutation = useCreateLoop()

  const setField =
    (key: keyof NewLoopFormState) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      setForm((f) => ({ ...f, [key]: e.target.value }))
      setErrors((err) => ({ ...err, [key]: undefined }))
    }

  const handleClose = () => {
    setForm(EMPTY_FORM)
    setErrors({})
    onClose()
  }

  const handleSubmit = () => {
    const newErrors: Partial<NewLoopFormState> = {}
    if (!form.prompt.trim()) newErrors.prompt = "Prompt is required"
    if (!form.interval.trim()) newErrors.interval = "Interval is required"
    const intervalSeconds = parseIntervalToSeconds(form.interval)
    if (intervalSeconds <= 0) newErrors.interval = "Invalid interval (e.g. 5m, 1h, 30s)"
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors)
      return
    }

    const payload: CreateLoopPayload = {
      prompt: form.prompt.trim(),
      interval_seconds: intervalSeconds,
      max_runs: form.maxRuns ? parseInt(form.maxRuns, 10) : 0,
      expires_at: expiresAtFromOption(form.expiresAfter),
    }

    createMutation.mutate(payload, { onSuccess: handleClose })
  }

  return (
    <Sheet open={open} onOpenChange={(v) => !v && handleClose()}>
      <SheetContent
        side="right"
        className="flex flex-col gap-0 p-0 data-[side=right]:!w-full data-[side=right]:sm:!w-[480px] data-[side=right]:sm:!max-w-[480px]"
      >
        <SheetHeader className="border-b border-b-muted px-6 py-5">
          <SheetTitle className="text-base">New Loop</SheetTitle>
          <SheetDescription className="text-xs">
            Schedule a repeating agent task.
          </SheetDescription>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="space-y-5 px-6 py-5">
            <Field label="Prompt" required error={errors.prompt}>
              <Textarea
                value={form.prompt}
                onChange={setField("prompt")}
                placeholder="Describe what the agent should do on each run…"
                rows={4}
                aria-invalid={!!errors.prompt}
              />
            </Field>

            <Field
              label="Interval"
              hint='How often to run. Examples: "30s", "5m", "1h", "24h".'
              error={errors.interval}
            >
              <Input
                value={form.interval}
                onChange={setField("interval")}
                placeholder="e.g. 5m"
                aria-invalid={!!errors.interval}
              />
            </Field>

            <Field
              label="Max Runs"
              hint="Maximum number of times to run. Leave blank or set 0 for unlimited."
            >
              <Input
                type="number"
                min={0}
                value={form.maxRuns}
                onChange={setField("maxRuns")}
                placeholder="0 = unlimited"
              />
            </Field>

            <Field label="Expires After" hint="Automatically stop the loop after this duration.">
              <Select
                value={form.expiresAfter}
                onValueChange={(v) =>
                  setForm((f) => ({ ...f, expiresAfter: v }))
                }
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1h">1 hour</SelectItem>
                  <SelectItem value="6h">6 hours</SelectItem>
                  <SelectItem value="24h">24 hours</SelectItem>
                  <SelectItem value="72h">72 hours</SelectItem>
                  <SelectItem value="never">Never</SelectItem>
                </SelectContent>
              </Select>
            </Field>
          </div>
        </div>

        <SheetFooter className="border-t border-t-muted px-6 py-4">
          <Button variant="ghost" onClick={handleClose} disabled={createMutation.isPending}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={createMutation.isPending}>
            {createMutation.isPending && (
              <IconLoader className="size-4 animate-spin" />
            )}
            Create Loop
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

// ─── Main page ────────────────────────────────────────────────────────────────

export function LoopsPage() {
  const { data: loops, isLoading, error } = useLoops()
  const deleteMutation = useDeleteLoop()
  const pauseMutation = usePauseLoop()
  const resumeMutation = useResumeLoop()

  const [createOpen, setCreateOpen] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<Loop | null>(null)

  const isActionPending =
    deleteMutation.isPending ||
    pauseMutation.isPending ||
    resumeMutation.isPending

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <PageHeader title="Loops">
        <Button size="sm" variant="outline" onClick={() => setCreateOpen(true)}>
          <IconPlus className="size-4" />
          New Loop
        </Button>
      </PageHeader>

      <div className="flex-1 overflow-auto px-6 py-4">
        {isLoading && (
          <div className="flex items-center justify-center py-12 text-muted-foreground">
            <IconLoader className="size-5 animate-spin mr-2" />
            <span className="text-sm">Loading loops…</span>
          </div>
        )}
        {error && (
          <div className="py-12 text-center text-sm text-destructive">
            Failed to load loops: {(error as Error).message}
          </div>
        )}
        {!isLoading && (
          <>
            {!loops || loops.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-20 gap-3 text-center">
                <IconRefresh className="size-10 text-muted-foreground/40" />
                <div>
                  <p className="text-sm font-medium">No loops configured</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Create a loop to schedule recurring agent tasks.
                  </p>
                </div>
                <Button size="sm" onClick={() => setCreateOpen(true)} className="mt-2">
                  <IconPlus className="size-4 mr-1" />
                  New Loop
                </Button>
              </div>
            ) : (
              <div className="rounded-lg border border-border/60 overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border/60 bg-muted/40">
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                          ID
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                          Prompt
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider whitespace-nowrap">
                          Interval
                        </th>
                        <th className="px-4 py-3 text-right text-xs font-semibold text-muted-foreground uppercase tracking-wider whitespace-nowrap">
                          Runs
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider whitespace-nowrap">
                          Last Run
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider whitespace-nowrap">
                          Next Run
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                          Status
                        </th>
                        <th className="px-4 py-3 text-right text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                          Actions
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border/40">
                      {loops.map((loop) => (
                        <tr
                          key={loop.id}
                          className="hover:bg-muted/20 transition-colors"
                        >
                          <td className="px-4 py-3">
                            <span className="font-mono text-xs text-muted-foreground">
                              {loop.id}
                            </span>
                          </td>
                          <td className="px-4 py-3 max-w-[260px]">
                            <span
                              className="text-sm"
                              title={loop.prompt}
                            >
                              {truncate(loop.prompt, 50)}
                            </span>
                          </td>
                          <td className="px-4 py-3">
                            <span className="font-mono text-xs">{loop.interval}</span>
                          </td>
                          <td className="px-4 py-3 text-right tabular-nums text-xs text-muted-foreground whitespace-nowrap">
                            {loop.run_count}
                            {loop.max_runs > 0 && (
                              <span className="text-muted-foreground/60">
                                {" / "}{loop.max_runs}
                              </span>
                            )}
                          </td>
                          <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                            {formatDate(loop.last_run)}
                          </td>
                          <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                            {loop.status === "active"
                              ? formatDate(loop.next_run)
                              : "—"}
                          </td>
                          <td className="px-4 py-3">
                            <StatusBadge status={loop.status} />
                          </td>
                          <td className="px-4 py-3">
                            <div className="flex items-center justify-end gap-1">
                              {loop.status === "active" && (
                                <Button
                                  size="icon-sm"
                                  variant="ghost"
                                  className="text-muted-foreground hover:text-foreground"
                                  disabled={isActionPending}
                                  onClick={() => pauseMutation.mutate(loop.id)}
                                  title="Pause loop"
                                >
                                  <IconPlayerPause className="size-4" />
                                </Button>
                              )}
                              {loop.status === "paused" && (
                                <Button
                                  size="icon-sm"
                                  variant="ghost"
                                  className="text-muted-foreground hover:text-foreground"
                                  disabled={isActionPending}
                                  onClick={() => resumeMutation.mutate(loop.id)}
                                  title="Resume loop"
                                >
                                  <IconPlayerPlay className="size-4" />
                                </Button>
                              )}
                              <Button
                                size="icon-sm"
                                variant="ghost"
                                className="text-muted-foreground hover:text-destructive"
                                disabled={isActionPending}
                                onClick={() => setPendingDelete(loop)}
                                title="Delete loop"
                              >
                                <IconTrash className="size-4" />
                              </Button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      <NewLoopSheet open={createOpen} onClose={() => setCreateOpen(false)} />

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(v) => !v && setPendingDelete(null)}
      >
        <AlertDialogContent size="sm">
          <AlertDialogHeader>
            <AlertDialogTitle>Delete loop?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete loop{" "}
              <strong className="font-mono">{pendingDelete?.id}</strong>. This
              action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel
              onClick={() => setPendingDelete(null)}
              disabled={deleteMutation.isPending}
            >
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => {
                if (pendingDelete) {
                  deleteMutation.mutate(pendingDelete.id, {
                    onSuccess: () => setPendingDelete(null),
                  })
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
