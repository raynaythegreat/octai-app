import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  type OAuthFlowState,
  type OAuthProvider,
  type OAuthProviderStatus,
  getOAuthFlow,
  getOAuthProviders,
  loginOAuth,
  logoutOAuth,
  pollOAuthFlow,
} from "@/api/oauth"
import {
  getModels,
  updateModel,
  testModelKey,
  getImageModels,
  updateImageModel,
  testImageModelKey,
  getVideoModels,
  updateVideoModel,
  testVideoModelKey,
  type ModelInfo,
} from "@/api/models"
import { toast } from "sonner"

type FlowWatchMode = "" | "status" | "poll"

const OAUTH_PROTOCOLS = new Set(["openai", "antigravity", "google-antigravity", "qwen", "minimax"])

function getProviderLabel(provider: OAuthProvider | ""): string {
  if (provider === "openai") return "OpenAI"
  if (provider === "anthropic") return "Anthropic"
  if (provider === "google-antigravity") return "Google Antigravity"
  if (provider === "qwen") return "Qwen"
  if (provider === "minimax") return "MiniMax"
  return ""
}

export function useCredentialsPage() {
  const { t } = useTranslation()
  const [providers, setProviders] = useState<OAuthProviderStatus[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  const [activeAction, setActiveAction] = useState("")
  const [activeFlow, setActiveFlow] = useState<OAuthFlowState | null>(null)
  const actionTokenRef = useRef(0)

  const [watchFlowID, setWatchFlowID] = useState("")
  const [watchMode, setWatchMode] = useState<FlowWatchMode>("")
  const [pollIntervalMs, setPollIntervalMs] = useState(2000)

  const [openAIToken, setOpenAIToken] = useState("")
  const [anthropicToken, setAnthropicToken] = useState("")

  const [logoutDialogOpen, setLogoutDialogOpen] = useState(false)
  const [logoutConfirmProvider, setLogoutConfirmProvider] = useState<
    OAuthProvider | ""
  >("")

  const [deviceSheetOpen, setDeviceSheetOpen] = useState(false)
  const [deviceFlow, setDeviceFlow] = useState<OAuthFlowState | null>(null)

  const [apiModels, setApiModels] = useState<ModelInfo[]>([])
  const [apiModelsLoading, setApiModelsLoading] = useState(false)

  const [imageModels, setImageModels] = useState<ModelInfo[]>([])
  const [videoModels, setVideoModels] = useState<ModelInfo[]>([])
  const [imageModelsLoading, setImageModelsLoading] = useState(false)
  const [videoModelsLoading, setVideoModelsLoading] = useState(false)

  const loadProviders = useCallback(async () => {
    try {
      const data = await getOAuthProviders()
      setProviders(data.providers)
      setError("")
    } catch (err) {
      setError(
        err instanceof Error ? err.message : t("credentials.errors.loadFailed"),
      )
    } finally {
      setLoading(false)
    }
  }, [t])

  const loadApiModels = useCallback(async () => {
    setApiModelsLoading(true)
    try {
      const data = await getModels()
      setApiModels(data.models)
    } catch {
      // silent — page still usable with OAuth providers
    } finally {
      setApiModelsLoading(false)
    }
  }, [])

  const loadImageModels = useCallback(async () => {
    setImageModelsLoading(true)
    try {
      const data = await getImageModels()
      setImageModels(data.models)
    } catch {
      // silent
    } finally {
      setImageModelsLoading(false)
    }
  }, [])

  const loadVideoModels = useCallback(async () => {
    setVideoModelsLoading(true)
    try {
      const data = await getVideoModels()
      setVideoModels(data.models)
    } catch {
      // silent
    } finally {
      setVideoModelsLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadProviders()
    void loadApiModels()
    void loadImageModels()
    void loadVideoModels()
  }, [loadProviders, loadApiModels, loadImageModels, loadVideoModels])

  useEffect(() => {
    if (!watchFlowID || !watchMode) {
      return
    }

    let canceled = false
    let timer: ReturnType<typeof setTimeout> | null = null

    const step = async () => {
      try {
        const flow =
          watchMode === "poll"
            ? await pollOAuthFlow(watchFlowID)
            : await getOAuthFlow(watchFlowID)

        if (canceled) {
          return
        }

        setActiveFlow(flow)
        setDeviceFlow((prev) =>
          prev?.flow_id === flow.flow_id ? { ...prev, ...flow } : prev,
        )

        if (flow.status === "pending") {
          timer = setTimeout(step, pollIntervalMs)
          return
        }

        if (watchMode === "poll") {
          setDeviceSheetOpen(false)
        }

        setWatchFlowID("")
        setWatchMode("")
        setActiveAction("")
        await loadProviders()
      } catch (err) {
        if (canceled) {
          return
        }
        setWatchFlowID("")
        setWatchMode("")
        setActiveAction("")
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.flowFailed"),
        )
      }
    }

    void step()

    return () => {
      canceled = true
      if (timer) {
        clearTimeout(timer)
      }
    }
  }, [loadProviders, pollIntervalMs, t, watchFlowID, watchMode])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const flowID = params.get("oauth_flow_id")
    if (!flowID) {
      return
    }

    setWatchFlowID(flowID)
    setWatchMode("status")
    setPollIntervalMs(700)

    window.history.replaceState({}, "", window.location.pathname)
  }, [])

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      const data = event.data as
        | { type?: string; flowId?: string; status?: string }
        | undefined
      if (!data || data.type !== "octai-oauth-result" || !data.flowId) {
        return
      }

      setWatchFlowID(data.flowId)
      setWatchMode("status")
      setPollIntervalMs(700)
    }

    window.addEventListener("message", onMessage)
    return () => window.removeEventListener("message", onMessage)
  }, [])

  const providersMap = useMemo(() => {
    const map = new Map<OAuthProvider, OAuthProviderStatus>()
    for (const item of providers) {
      map.set(item.provider, item)
    }
    return map
  }, [providers])

  const openaiStatus = providersMap.get("openai")
  const anthropicStatus = providersMap.get("anthropic")
  const antigravityStatus = providersMap.get("google-antigravity")
  const qwenStatus = providersMap.get("qwen")
  const minimaxStatus = providersMap.get("minimax")

  const bumpActionToken = useCallback(() => {
    actionTokenRef.current += 1
    return actionTokenRef.current
  }, [])

  const isActionTokenCurrent = useCallback((token: number) => {
    return actionTokenRef.current === token
  }, [])

  const startBrowserOAuth = useCallback(
    async (provider: OAuthProvider) => {
      const actionToken = bumpActionToken()
      setActiveAction(`${provider}:browser`)
      setError("")

      const authTab = window.open("", "_blank")
      if (!authTab) {
        if (!isActionTokenCurrent(actionToken)) {
          return
        }
        setActiveAction("")
        setError(t("credentials.errors.popupBlocked"))
        return
      }

      try {
        const resp = await loginOAuth({ provider, method: "browser" })
        if (!isActionTokenCurrent(actionToken)) {
          authTab.close()
          return
        }
        if (!resp.auth_url || !resp.flow_id) {
          throw new Error(t("credentials.errors.invalidBrowserResponse"))
        }

        authTab.location.href = resp.auth_url

        setActiveFlow({
          flow_id: resp.flow_id,
          provider,
          method: "browser",
          status: "pending",
          expires_at: resp.expires_at,
        })
        setWatchFlowID(resp.flow_id)
        setWatchMode("status")
        setPollIntervalMs(2000)
      } catch (err) {
        if (!isActionTokenCurrent(actionToken)) {
          authTab.close()
          return
        }
        authTab.close()
        setActiveAction("")
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.loginFailed"),
        )
      }
    },
    [bumpActionToken, isActionTokenCurrent, t],
  )

  const startDeviceCode = useCallback(
    async (provider: OAuthProvider) => {
      const actionToken = bumpActionToken()
      setActiveAction(`${provider}:device`)
      setError("")

      try {
        const resp = await loginOAuth({
          provider,
          method: "device_code",
        })
        if (!isActionTokenCurrent(actionToken)) {
          return
        }
        if (!resp.flow_id || !resp.user_code || !resp.verify_url) {
          throw new Error(t("credentials.errors.invalidDeviceResponse"))
        }

        const flow: OAuthFlowState = {
          flow_id: resp.flow_id,
          provider,
          method: "device_code",
          status: "pending",
          user_code: resp.user_code,
          verify_url: resp.verify_url,
          interval: resp.interval,
          expires_at: resp.expires_at,
        }

        setDeviceFlow(flow)
        setDeviceSheetOpen(true)
        setActiveFlow(flow)
        setWatchFlowID(resp.flow_id)
        setWatchMode("poll")
        setPollIntervalMs(Math.max(1000, (resp.interval ?? 5) * 1000))
      } catch (err) {
        if (!isActionTokenCurrent(actionToken)) {
          return
        }
        setActiveAction("")
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.loginFailed"),
        )
      }
    },
    [bumpActionToken, isActionTokenCurrent, t],
  )

  const startOpenAIDeviceCode = useCallback(() => {
    return startDeviceCode("openai")
  }, [startDeviceCode])

  const saveToken = useCallback(
    async (provider: OAuthProvider, token: string) => {
      const actionID = `${provider}:token`
      setActiveAction(actionID)
      setError("")

      try {
        await loginOAuth({ provider, method: "token", token })
        if (provider === "openai") {
          setOpenAIToken("")
        }
        if (provider === "anthropic") {
          setAnthropicToken("")
        }
        await loadProviders()
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.loginFailed"),
        )
      } finally {
        setActiveAction("")
      }
    },
    [loadProviders, t],
  )

  const doLogout = useCallback(
    async (provider: OAuthProvider) => {
      const actionID = `${provider}:logout`
      setActiveAction(actionID)
      setError("")

      try {
        await logoutOAuth(provider)
        await loadProviders()
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.logoutFailed"),
        )
      } finally {
        setActiveAction("")
      }
    },
    [loadProviders, t],
  )

  const askLogout = useCallback((provider: OAuthProvider) => {
    setLogoutConfirmProvider(provider)
    setLogoutDialogOpen(true)
  }, [])

  const handleConfirmLogout = useCallback(async () => {
    if (!logoutConfirmProvider) {
      return
    }
    await doLogout(logoutConfirmProvider)
    setLogoutDialogOpen(false)
    setLogoutConfirmProvider("")
  }, [doLogout, logoutConfirmProvider])

  const handleLogoutDialogOpenChange = useCallback((open: boolean) => {
    setLogoutDialogOpen(open)
    if (!open) {
      setLogoutConfirmProvider("")
    }
  }, [])

  const handleDeviceSheetOpenChange = useCallback(
    (open: boolean) => {
      setDeviceSheetOpen(open)
      if (open) {
        return
      }

      if (watchMode === "poll") {
        setWatchFlowID("")
        setWatchMode("")
        if (activeAction.endsWith(":device")) {
          setActiveAction("")
        }
      }

      setDeviceFlow(null)
      if (
        activeFlow?.method === "device_code" &&
        activeFlow.status === "pending"
      ) {
        setActiveFlow(null)
      }
    },
    [activeAction, activeFlow, watchMode],
  )

  const stopLoading = useCallback(() => {
    bumpActionToken()
    setWatchFlowID("")
    setWatchMode("")
    setActiveAction("")
    setDeviceSheetOpen(false)
    setDeviceFlow(null)
    setActiveFlow((prev) => (prev?.status === "pending" ? null : prev))
  }, [bumpActionToken])

  const logoutProviderLabel = getProviderLabel(logoutConfirmProvider)

  const flowHint = useMemo(() => {
    if (!activeFlow) {
      return ""
    }
    if (activeFlow.status === "pending") {
      return t("credentials.flow.pending")
    }
    if (activeFlow.status === "success") {
      return t("credentials.flow.success")
    }
    if (activeFlow.status === "expired") {
      return t("credentials.flow.expired")
    }
    return activeFlow.error || t("credentials.flow.error")
  }, [activeFlow, t])

  const saveApiKey = useCallback(
    async (index: number, key: string) => {
      const model = apiModels.find((m) => m.index === index)
      if (!model) return
      try {
        await updateModel(index, {
          model_name: model.model_name,
          model: model.model,
          api_base: model.api_base ?? "",
          api_key: key,
        })
        toast.success(t("credentials.save.success"))
        await loadApiModels()
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : t("credentials.errors.saveFailed"),
        )
      }
    },
    [apiModels, loadApiModels, t],
  )

  const deleteApiKey = useCallback(
    async (index: number) => {
      const model = apiModels.find((m) => m.index === index)
      if (!model) return
      try {
        await updateModel(index, {
          model_name: model.model_name,
          model: model.model,
          api_base: model.api_base ?? "",
          api_key: "REMOVED",
        })
        toast.success(t("credentials.delete.success"))
        await loadApiModels()
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : t("credentials.errors.logoutFailed"),
        )
      }
    },
    [apiModels, loadApiModels, t],
  )

  const testApiKey = useCallback(
    async (index: number): Promise<{ success: boolean; models?: string[] }> => {
      try {
        const result = await testModelKey(index)
        if (result.success) {
          toast.success(t("credentials.test.success"))
        } else {
          toast.error(
            t("credentials.test.failure", { error: result.error ?? "Unknown error" }),
          )
        }
        return result
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Test failed"
        toast.error(t("credentials.test.failure", { error: msg }))
        return { success: false }
      }
    },
    [t],
  )

  const saveImageApiKey = useCallback(
    async (index: number, key: string) => {
      const model = imageModels.find((m) => m.index === index)
      if (!model) return
      try {
        await updateImageModel(index, {
          model_name: model.model_name,
          model: model.model,
          api_base: model.api_base ?? "",
          api_key: key,
        })
        toast.success(t("credentials.save.success"))
        await loadImageModels()
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : t("credentials.errors.saveFailed"),
        )
      }
    },
    [imageModels, loadImageModels, t],
  )

  const deleteImageApiKey = useCallback(
    async (index: number) => {
      const model = imageModels.find((m) => m.index === index)
      if (!model) return
      try {
        await updateImageModel(index, {
          model_name: model.model_name,
          model: model.model,
          api_base: model.api_base ?? "",
          api_key: "REMOVED",
        })
        toast.success(t("credentials.delete.success"))
        await loadImageModels()
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : t("credentials.errors.logoutFailed"),
        )
      }
    },
    [imageModels, loadImageModels, t],
  )

  const testImageApiKey = useCallback(
    async (index: number): Promise<{ success: boolean; models?: string[] }> => {
      try {
        const result = await testImageModelKey(index)
        if (result.success) {
          toast.success(t("credentials.test.success"))
        } else {
          toast.error(
            t("credentials.test.failure", { error: result.error ?? "Unknown error" }),
          )
        }
        return result
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Test failed"
        toast.error(t("credentials.test.failure", { error: msg }))
        return { success: false }
      }
    },
    [t],
  )

  const saveVideoApiKey = useCallback(
    async (index: number, key: string) => {
      const model = videoModels.find((m) => m.index === index)
      if (!model) return
      try {
        await updateVideoModel(index, {
          model_name: model.model_name,
          model: model.model,
          api_base: model.api_base ?? "",
          api_key: key,
        })
        toast.success(t("credentials.save.success"))
        await loadVideoModels()
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : t("credentials.errors.saveFailed"),
        )
      }
    },
    [videoModels, loadVideoModels, t],
  )

  const deleteVideoApiKey = useCallback(
    async (index: number) => {
      const model = videoModels.find((m) => m.index === index)
      if (!model) return
      try {
        await updateVideoModel(index, {
          model_name: model.model_name,
          model: model.model,
          api_base: model.api_base ?? "",
          api_key: "REMOVED",
        })
        toast.success(t("credentials.delete.success"))
        await loadVideoModels()
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : t("credentials.errors.logoutFailed"),
        )
      }
    },
    [videoModels, loadVideoModels, t],
  )

  const testVideoApiKey = useCallback(
    async (index: number): Promise<{ success: boolean; models?: string[] }> => {
      try {
        const result = await testVideoModelKey(index)
        if (result.success) {
          toast.success(t("credentials.test.success"))
        } else {
          toast.error(
            t("credentials.test.failure", { error: result.error ?? "Unknown error" }),
          )
        }
        return result
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Test failed"
        toast.error(t("credentials.test.failure", { error: msg }))
        return { success: false }
      }
    },
    [t],
  )

  const providerModelGroups = useMemo(() => {
    const groups = new Map<string, ModelInfo[]>()
    for (const m of apiModels) {
      if (m.is_virtual) continue
      // Skip OAuth providers (they have dedicated cards)
      const protocol = m.model.split("/")[0].toLowerCase()
      if (OAUTH_PROTOCOLS.has(protocol)) continue
      // Skip models with no api_base (CLI-based) and no protocol prefix
      if (!m.api_base && protocol === "openai") continue
      const existing = groups.get(protocol) ?? []
      existing.push(m)
      groups.set(protocol, existing)
    }
    return groups
  }, [apiModels])

  return {
    loading,
    error,
    activeAction,
    activeFlow,
    flowHint,
    openAIToken,
    anthropicToken,
    openaiStatus,
    anthropicStatus,
    antigravityStatus,
    qwenStatus,
    minimaxStatus,
    logoutDialogOpen,
    logoutConfirmProvider,
    logoutProviderLabel,
    deviceSheetOpen,
    deviceFlow,
    setOpenAIToken,
    setAnthropicToken,
    startBrowserOAuth,
    startOpenAIDeviceCode,
    startDeviceCode,
    stopLoading,
    saveToken,
    askLogout,
    handleConfirmLogout,
    handleLogoutDialogOpenChange,
    handleDeviceSheetOpenChange,
    apiModels,
    apiModelsLoading,
    providerModelGroups,
    saveApiKey,
    deleteApiKey,
    testApiKey,
    loadApiModels,
    imageModels,
    videoModels,
    imageModelsLoading,
    videoModelsLoading,
    saveImageApiKey,
    deleteImageApiKey,
    testImageApiKey,
    saveVideoApiKey,
    deleteVideoApiKey,
    testVideoApiKey,
    loadImageModels,
    loadVideoModels,
  }
}
