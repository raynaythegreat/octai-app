import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { toast } from "sonner"

import {
  getGatewayStatus,
  restartGateway,
  startGateway,
} from "@/api/gateway"
import { type ModelInfo, getAutoRouting, getModels, setAutoRouting, setDefaultModel } from "@/api/models"

interface UseChatModelsOptions {
  isConnected: boolean
}

function isLocalModel(model: ModelInfo): boolean {
  const isLocalHostBase = Boolean(
    model.api_base?.includes("localhost") ||
    model.api_base?.includes("127.0.0.1"),
  )

  return (
    model.auth_method === "local" || (!model.auth_method && isLocalHostBase)
  )
}

export function useChatModels({ isConnected }: UseChatModelsOptions) {
  const [modelList, setModelList] = useState<ModelInfo[]>([])
  const [defaultModelName, setDefaultModelName] = useState("")
  const [isAutoMode, setIsAutoMode] = useState(true)
  const setDefaultRequestIdRef = useRef(0)
  const autoSelectingRef = useRef(false)

  const loadModels = useCallback(async () => {
    try {
      const data = await getModels()
      setModelList(data.models)
      if (
        data.models.some(
          (m) =>
            m.model_name === data.default_model &&
            m.chat_enabled &&
            m.configured,
        )
      ) {
        setDefaultModelName(data.default_model)
      } else {
        setDefaultModelName("")
      }
    } catch {
      // silently fail
    }
  }, [])

  const loadAutoRouting = useCallback(async () => {
    try {
      const data = await getAutoRouting()
      setIsAutoMode(data.enabled)
    } catch {
      // silently fail
    }
  }, [])

  useEffect(() => {
    const timerId = setTimeout(() => {
      void loadModels()
      void loadAutoRouting()
    }, 0)

    return () => clearTimeout(timerId)
  }, [isConnected, loadModels, loadAutoRouting])

  const syncGatewayForModelChange = useCallback(async () => {
    try {
      const status = await getGatewayStatus()
      if (status.gateway_status === "running") {
        await restartGateway()
        return
      }
      if (
        (status.gateway_status === "stopped" || status.gateway_status === "error") &&
        status.gateway_start_allowed
      ) {
        await startGateway()
      }
    } catch (err) {
      console.error("Failed to sync gateway after model change:", err)
    }
  }, [])

  const toggleAutoMode = useCallback(async (enabled: boolean) => {
    try {
      await setAutoRouting({ enabled })
      setIsAutoMode(enabled)
      await loadModels()
      await loadAutoRouting()
      await syncGatewayForModelChange()
      toast(enabled ? "Auto mode enabled" : "Auto mode disabled")
    } catch (err) {
      console.error("Failed to toggle auto mode:", err)
    }
  }, [loadAutoRouting, loadModels, syncGatewayForModelChange])

  const handleSetDefault = useCallback(
    async (modelName: string) => {
      if (modelName === defaultModelName) return
      const requestId = ++setDefaultRequestIdRef.current

      try {
        await setDefaultModel(modelName)
        const data = await getModels()
        if (requestId !== setDefaultRequestIdRef.current) {
          return
        }

        setModelList(data.models)
        if (
          data.models.some(
            (m) =>
              m.model_name === data.default_model &&
              m.chat_enabled &&
              m.configured,
          )
        ) {
          setDefaultModelName(data.default_model)
        } else {
          setDefaultModelName("")
        }
        await syncGatewayForModelChange()
      } catch (err) {
        console.error("Failed to set default model:", err)
      }
    },
    [defaultModelName, syncGatewayForModelChange],
  )

  const hasSavedModels = useMemo(() => modelList.length > 0, [modelList])

  const chatEnabledModels = useMemo(
    () => modelList.filter((m) => m.chat_enabled),
    [modelList],
  )

  const hasConfiguredModels = useMemo(
    () => modelList.some((m) => m.configured),
    [modelList],
  )

  const availableModels = useMemo(
    () => chatEnabledModels.filter((m) => m.configured),
    [chatEnabledModels],
  )

  const hasChatEnabledModels = useMemo(
    () => chatEnabledModels.length > 0,
    [chatEnabledModels],
  )

  const oauthModels = useMemo(
    () => availableModels.filter((m) => m.auth_method === "oauth"),
    [availableModels],
  )

  const localModels = useMemo(
    () => availableModels.filter((m) => isLocalModel(m)),
    [availableModels],
  )

  const apiKeyModels = useMemo(
    () =>
      availableModels.filter(
        (m) => m.auth_method !== "oauth" && !isLocalModel(m),
      ),
    [availableModels],
  )

  const hasAvailableModels = useMemo(
    () => availableModels.length > 0,
    [availableModels],
  )

  useEffect(() => {
    if (!isAutoMode || autoSelectingRef.current) {
      return
    }
    const hasValidDefault = modelList.some(
      (m) => m.model_name === defaultModelName && m.chat_enabled && m.configured,
    )
    if (hasValidDefault || availableModels.length === 0) {
      return
    }

    autoSelectingRef.current = true
    void handleSetDefault(availableModels[0].model_name).finally(() => {
      autoSelectingRef.current = false
    })
  }, [availableModels, defaultModelName, handleSetDefault, isAutoMode, modelList])

  useEffect(() => {
    if (!modelList.some((m) => m.model_name === defaultModelName && m.chat_enabled && m.configured)) {
      setDefaultModelName("")
    }
  }, [defaultModelName, modelList])

  return {
    defaultModelName,
    hasSavedModels,
    hasConfiguredModels,
    hasChatEnabledModels,
    hasAvailableModels,
    apiKeyModels,
    oauthModels,
    localModels,
    handleSetDefault,
    isAutoMode,
    toggleAutoMode,
  }
}
