import { refreshGatewayState } from "@/store/gateway"

// API client for model list management.

export interface ModelInfo {
  index: number
  model_name: string
  model: string
  api_base?: string
  api_key: string
  proxy?: string
  auth_method?: string
  // Advanced fields
  connect_mode?: string
  workspace?: string
  rpm?: number
  max_tokens_field?: string
  request_timeout?: number
  thinking_level?: string
  extra_body?: Record<string, unknown>
  // Meta
  configured: boolean
  available?: boolean
  chat_enabled: boolean
  is_default: boolean
  is_virtual: boolean
}

interface ModelsListResponse {
  models: ModelInfo[]
  total: number
  default_model: string
}

interface ModelActionResponse {
  status: string
  index?: number
  default_model?: string
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getModels(params?: { configured_only?: boolean }): Promise<ModelsListResponse> {
  const query = params?.configured_only ? "?configured_only=true" : ""
  return request<ModelsListResponse>(`/api/models${query}`)
}

export async function addModel(
  model: Partial<ModelInfo>,
): Promise<ModelActionResponse> {
  return request<ModelActionResponse>("/api/models", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(model),
  })
}

export async function updateModel(
  index: number,
  model: Partial<ModelInfo>,
): Promise<ModelActionResponse> {
  return request<ModelActionResponse>(`/api/models/${index}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(model),
  })
}

export async function deleteModel(index: number): Promise<ModelActionResponse> {
  return request<ModelActionResponse>(`/api/models/${index}`, {
    method: "DELETE",
  })
}

export async function setDefaultModel(
  modelName: string,
): Promise<ModelActionResponse> {
  const response = await request<ModelActionResponse>("/api/models/default", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ model_name: modelName }),
  })

  await refreshGatewayState()
  return response
}

export interface TestModelKeyResult {
  success: boolean
  error?: string
  models?: string[]
}

export async function testModelKey(index: number): Promise<TestModelKeyResult> {
  return request<TestModelKeyResult>(`/api/models/${index}/test`, {
    method: "POST",
  })
}

export async function setModelChatEnabled(
  index: number,
  enabled: boolean,
): Promise<{ status: string; chat_enabled: boolean }> {
  return request<{ status: string; chat_enabled: boolean }>(
    `/api/models/${index}/chat-enabled`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    },
  )
}

export async function getImageModels(): Promise<{ models: ModelInfo[]; total: number }> {
  const res = await fetch("/api/image-models")
  if (!res.ok) throw new Error(`Failed to fetch image models: ${res.status}`)
  return res.json() as Promise<{ models: ModelInfo[]; total: number }>
}

export async function updateImageModel(
  index: number,
  data: Partial<ModelInfo>,
): Promise<void> {
  const res = await fetch(`/api/image-models/${index}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`Failed to update image model: ${res.status}`)
}

export async function testImageModelKey(
  index: number,
): Promise<TestModelKeyResult> {
  const res = await fetch(`/api/image-models/${index}/test`, { method: "POST" })
  if (!res.ok) throw new Error(`Failed to test image model key: ${res.status}`)
  return res.json() as Promise<TestModelKeyResult>
}

export async function setImageModelChatEnabled(
  index: number,
  enabled: boolean,
): Promise<{ status: string; chat_enabled: boolean }> {
  return request<{ status: string; chat_enabled: boolean }>(
    `/api/image-models/${index}/chat-enabled`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    },
  )
}

export async function getVideoModels(): Promise<{ models: ModelInfo[]; total: number }> {
  const res = await fetch("/api/video-models")
  if (!res.ok) throw new Error(`Failed to fetch video models: ${res.status}`)
  return res.json() as Promise<{ models: ModelInfo[]; total: number }>
}

export async function updateVideoModel(
  index: number,
  data: Partial<ModelInfo>,
): Promise<void> {
  const res = await fetch(`/api/video-models/${index}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`Failed to update video model: ${res.status}`)
}

export async function testVideoModelKey(
  index: number,
): Promise<TestModelKeyResult> {
  const res = await fetch(`/api/video-models/${index}/test`, { method: "POST" })
  if (!res.ok) throw new Error(`Failed to test video model key: ${res.status}`)
  return res.json() as Promise<TestModelKeyResult>
}

export async function setVideoModelChatEnabled(
  index: number,
  enabled: boolean,
): Promise<{ status: string; chat_enabled: boolean }> {
  return request<{ status: string; chat_enabled: boolean }>(
    `/api/video-models/${index}/chat-enabled`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    },
  )
}

// Rotate API key functions
export async function rotateModelKey(
  index: number,
  newApiKey: string,
): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/models/${index}/rotate-key`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ new_api_key: newApiKey }),
  })
}

export async function rotateImageModelKey(
  index: number,
  newApiKey: string,
): Promise<{ status: string }> {
  const res = await fetch(`/api/image-models/${index}/rotate-key`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ new_api_key: newApiKey }),
  })
  if (!res.ok) throw new Error(`Failed to rotate image model key: ${res.status}`)
  return res.json() as Promise<{ status: string }>
}

export async function rotateVideoModelKey(
  index: number,
  newApiKey: string,
): Promise<{ status: string }> {
  const res = await fetch(`/api/video-models/${index}/rotate-key`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ new_api_key: newApiKey }),
  })
  if (!res.ok) throw new Error(`Failed to rotate video model key: ${res.status}`)
  return res.json() as Promise<{ status: string }>
}

// Delete image/video model functions
export async function deleteImageModel(index: number): Promise<void> {
  const res = await fetch(`/api/image-models/${index}`, { method: "DELETE" })
  if (!res.ok) throw new Error(`Failed to delete image model: ${res.status}`)
}

export async function deleteVideoModel(index: number): Promise<void> {
  const res = await fetch(`/api/video-models/${index}`, { method: "DELETE" })
  if (!res.ok) throw new Error(`Failed to delete video model: ${res.status}`)
}

// Add image/video model form data type
export interface ModelFormData {
  model_name: string
  model: string
  api_base?: string
  api_key?: string
}

export async function addImageModel(model: ModelFormData): Promise<void> {
  const res = await fetch("/api/image-models", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(model),
  })
  if (!res.ok) throw new Error(`Failed to add image model: ${res.status}`)
}

export async function addVideoModel(model: ModelFormData): Promise<void> {
  const res = await fetch("/api/video-models", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(model),
  })
  if (!res.ok) throw new Error(`Failed to add video model: ${res.status}`)
}

export async function getImageModelsFiltered(params?: { configured_only?: boolean }): Promise<{ models: ModelInfo[]; total: number }> {
  const query = params?.configured_only ? "?configured_only=true" : ""
  const res = await fetch(`/api/image-models${query}`)
  if (!res.ok) throw new Error(`Failed to fetch image models: ${res.status}`)
  return res.json() as Promise<{ models: ModelInfo[]; total: number }>
}

export async function getVideoModelsFiltered(params?: { configured_only?: boolean }): Promise<{ models: ModelInfo[]; total: number }> {
  const query = params?.configured_only ? "?configured_only=true" : ""
  const res = await fetch(`/api/video-models${query}`)
  if (!res.ok) throw new Error(`Failed to fetch video models: ${res.status}`)
  return res.json() as Promise<{ models: ModelInfo[]; total: number }>
}

export interface RoutingConfig {
  enabled: boolean
  light_model: string
  threshold: number
}

export interface ModelFallbackConfig {
  fallback_model: string
}

export async function getAutoRouting(): Promise<RoutingConfig> {
  return request<RoutingConfig>("/api/models/auto")
}

export async function setAutoRouting(
  routingConfig: Partial<RoutingConfig>,
): Promise<{ status: string }> {
  return request<{ status: string }>("/api/models/auto", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(routingConfig),
  })
}

export async function getModelFallback(): Promise<ModelFallbackConfig> {
  return request<ModelFallbackConfig>("/api/models/fallback")
}

export async function setModelFallback(
  modelName: string,
): Promise<{ status: string; fallback_model: string }> {
  const response = await request<{ status: string; fallback_model: string }>(
    "/api/models/fallback",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ model_name: modelName }),
    },
  )

  await refreshGatewayState()
  return response
}

export type { ModelsListResponse, ModelActionResponse }
