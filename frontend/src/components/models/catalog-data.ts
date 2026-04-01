export type CatalogCategory =
  | "Popular"
  | "Reasoning"
  | "Fast Inference"
  | "Other Cloud"
  | "Local"
  | "Gateway"

export interface CatalogModel {
  name: string
  id: string
  desc: string
}

export interface CatalogProvider {
  provider: string
  providerKey: string
  category: CatalogCategory
  docsUrl?: string
  models: CatalogModel[]
}

export const MODEL_CATALOG: CatalogProvider[] = [
  // Popular
  {
    provider: "Anthropic",
    providerKey: "anthropic",
    category: "Popular",
    docsUrl: "https://docs.anthropic.com/en/api/getting-started",
    models: [
      {
        name: "claude-opus-4-6",
        id: "anthropic/claude-opus-4-6",
        desc: "Most powerful Claude — complex reasoning",
      },
      {
        name: "claude-sonnet-4.6",
        id: "anthropic/claude-sonnet-4-6",
        desc: "Best balance of intelligence and speed",
      },
      {
        name: "claude-haiku-4-5",
        id: "anthropic/claude-haiku-4-5-20251001",
        desc: "Fastest Claude — lightweight tasks",
      },
    ],
  },
  {
    provider: "OpenAI",
    providerKey: "openai",
    category: "Popular",
    docsUrl: "https://platform.openai.com/docs/overview",
    models: [
      {
        name: "gpt-5.4",
        id: "openai/gpt-5.4",
        desc: "Flagship — complex reasoning, 1M context",
      },
      {
        name: "gpt-5",
        id: "openai/gpt-5",
        desc: "Previous flagship — still highly capable",
      },
      {
        name: "gpt-5.4-mini",
        id: "openai/gpt-5.4-mini",
        desc: "Fast, efficient — high-volume workloads",
      },
      {
        name: "gpt-5.4-nano",
        id: "openai/gpt-5.4-nano",
        desc: "Cheapest GPT-5.4-class — classification, sub-agents",
      },
      {
        name: "o3",
        id: "openai/o3",
        desc: "Advanced reasoning model",
      },
      {
        name: "o4-mini",
        id: "openai/o4-mini",
        desc: "Fast reasoning",
      },
    ],
  },
  {
    provider: "Google Gemini",
    providerKey: "gemini",
    category: "Popular",
    docsUrl: "https://ai.google.dev/gemini-api/docs",
    models: [
      {
        name: "gemini-2.5-pro",
        id: "gemini/gemini-2.5-pro",
        desc: "Most advanced Gemini — complex tasks",
      },
      {
        name: "gemini-2.5-flash",
        id: "gemini/gemini-2.5-flash",
        desc: "Best price-performance reasoning",
      },
      {
        name: "gemini-2.5-flash-lite",
        id: "gemini/gemini-2.5-flash-lite",
        desc: "Fastest, most budget-friendly",
      },
      {
        name: "gemini-3-flash-preview",
        id: "gemini/gemini-3-flash-preview",
        desc: "Next-gen Gemini 3 Flash (preview)",
      },
    ],
  },
  // Reasoning
  {
    provider: "DeepSeek",
    providerKey: "deepseek",
    category: "Reasoning",
    docsUrl: "https://api-docs.deepseek.com/",
    models: [
      {
        name: "deepseek-chat",
        id: "deepseek/deepseek-chat",
        desc: "DeepSeek V3.2 — coding, math, general",
      },
      {
        name: "deepseek-reasoner",
        id: "deepseek/deepseek-reasoner",
        desc: "DeepSeek R1 — chain-of-thought reasoning",
      },
    ],
  },
  {
    provider: "xAI Grok",
    providerKey: "grok",
    category: "Reasoning",
    docsUrl: "https://docs.x.ai/docs",
    models: [
      {
        name: "grok-4-reasoning",
        id: "grok/grok-4.20-0309-reasoning",
        desc: "Grok 4 — flagship reasoning, 2M context",
      },
      {
        name: "grok-4-non-reasoning",
        id: "grok/grok-4.20-0309-non-reasoning",
        desc: "Grok 4 — standard, fastest response",
      },
      {
        name: "grok-4-fast",
        id: "grok/grok-4-1-fast-reasoning",
        desc: "Grok 4 Fast — speed-optimized",
      },
    ],
  },
  // Fast Inference
  {
    provider: "Groq",
    providerKey: "groq",
    category: "Fast Inference",
    docsUrl: "https://console.groq.com/docs/openai",
    models: [
      {
        name: "llama-3.3-70b-versatile",
        id: "groq/llama-3.3-70b-versatile",
        desc: "Ultra-fast 70B",
      },
      {
        name: "llama-3.1-8b-instant",
        id: "groq/llama-3.1-8b-instant",
        desc: "Ultra-fast 8B — instant",
      },
      {
        name: "llama-4-scout-17b",
        id: "groq/meta-llama/llama-4-scout-17b-16e-instruct",
        desc: "Llama 4 Scout — multimodal, 10M context",
      },
      {
        name: "qwen3-32b",
        id: "groq/qwen/qwen3-32b",
        desc: "Qwen 3 32B — strong reasoning",
      },
    ],
  },
  {
    provider: "Cerebras",
    providerKey: "cerebras",
    category: "Fast Inference",
    docsUrl: "https://inference-docs.cerebras.ai/introduction",
    models: [
      {
        name: "cerebras-gpt-oss-120b",
        id: "cerebras/gpt-oss-120b",
        desc: "GPT OSS 120B — fastest at 2,304 tok/s",
      },
      {
        name: "cerebras-llama-3.1-8b",
        id: "cerebras/llama3.1-8b",
        desc: "8B ultra-fast",
      },
    ],
  },
  // Other Cloud
  {
    provider: "Mistral AI",
    providerKey: "mistral",
    category: "Other Cloud",
    docsUrl: "https://docs.mistral.ai/",
    models: [
      {
        name: "mistral-large",
        id: "mistral/mistral-large-latest",
        desc: "Most capable Mistral",
      },
      {
        name: "mistral-medium",
        id: "mistral/mistral-medium-latest",
        desc: "Mistral Medium 3.1 — balanced",
      },
      {
        name: "mistral-small",
        id: "mistral/mistral-small-latest",
        desc: "Efficient and affordable",
      },
      {
        name: "codestral",
        id: "mistral/codestral-latest",
        desc: "Code specialist",
      },
      {
        name: "devstral-small",
        id: "mistral/devstral-small-2-25-12",
        desc: "Devstral Small 2 — beats Qwen 3 Coder",
      },
    ],
  },
  {
    provider: "Together AI",
    providerKey: "together",
    category: "Other Cloud",
    docsUrl: "https://docs.together.ai/docs/introduction",
    models: [
      {
        name: "together-llama-4-maverick",
        id: "together/meta-llama/Llama-4-Maverick-17B-128E-Instruct",
        desc: "Llama 4 Maverick — best multimodal open model",
      },
      {
        name: "together-deepseek-r1",
        id: "together/deepseek-ai/DeepSeek-R1",
        desc: "DeepSeek R1 reasoning",
      },
      {
        name: "together-kimi-k2",
        id: "together/moonshotai/Kimi-K2-Instruct",
        desc: "Kimi K2 — 262K context",
      },
      {
        name: "together-qwen3",
        id: "together/Qwen/Qwen3-235B-A22B-Instruct-2507-FP8",
        desc: "Qwen 3 235B — top open reasoning model",
      },
    ],
  },
  {
    provider: "Perplexity",
    providerKey: "perplexity",
    category: "Other Cloud",
    docsUrl: "https://docs.perplexity.ai/",
    models: [
      {
        name: "sonar",
        id: "perplexity/sonar",
        desc: "Lightweight web search, 128K context",
      },
      {
        name: "sonar-pro",
        id: "perplexity/sonar-pro",
        desc: "Advanced search, 2x more sources",
      },
      {
        name: "sonar-reasoning",
        id: "perplexity/sonar-reasoning",
        desc: "Real-time reasoning with web search",
      },
      {
        name: "sonar-reasoning-pro",
        id: "perplexity/sonar-reasoning-pro",
        desc: "Premium reasoning — DeepSeek-R1 powered",
      },
    ],
  },
  // Local
  {
    provider: "Ollama",
    providerKey: "ollama",
    category: "Local",
    docsUrl: "https://ollama.com/library",
    models: [
      {
        name: "llama4",
        id: "ollama/llama4",
        desc: "Llama 4 Scout — 17B MoE, 10M context",
      },
      {
        name: "qwen3",
        id: "ollama/qwen3",
        desc: "Qwen 3 — strong multilingual reasoning",
      },
      {
        name: "llama3.2",
        id: "ollama/llama3.2",
        desc: "Meta Llama 3.2 — local",
      },
      {
        name: "qwen2.5",
        id: "ollama/qwen2.5",
        desc: "Alibaba Qwen 2.5 — local",
      },
      {
        name: "codellama",
        id: "ollama/codellama",
        desc: "Code specialist — local",
      },
      {
        name: "phi3",
        id: "ollama/phi3",
        desc: "Microsoft Phi-3 — small but capable",
      },
      {
        name: "gemma2",
        id: "ollama/gemma2",
        desc: "Google Gemma 2 — local",
      },
    ],
  },
  // Gateway
  {
    provider: "OpenRouter",
    providerKey: "openrouter",
    category: "Gateway",
    docsUrl: "https://openrouter.ai/docs",
    models: [
      {
        name: "openrouter-auto",
        id: "openrouter/auto",
        desc: "Auto-routes to best model",
      },
      {
        name: "openrouter-claude-sonnet",
        id: "openrouter/anthropic/claude-sonnet-4-6",
        desc: "Claude via OpenRouter",
      },
      {
        name: "openrouter-gpt-5.4",
        id: "openrouter/openai/gpt-5.4",
        desc: "GPT-5.4 via OpenRouter",
      },
    ],
  },
]

export const CATALOG_CATEGORIES: CatalogCategory[] = [
  "Popular",
  "Reasoning",
  "Fast Inference",
  "Other Cloud",
  "Local",
  "Gateway",
]

export const CATEGORY_COLORS: Record<CatalogCategory, string> = {
  Popular: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  Reasoning: "bg-purple-500/10 text-purple-600 dark:text-purple-400",
  "Fast Inference": "bg-orange-500/10 text-orange-600 dark:text-orange-400",
  "Other Cloud": "bg-teal-500/10 text-teal-600 dark:text-teal-400",
  Local: "bg-green-500/10 text-green-600 dark:text-green-400",
  Gateway: "bg-gray-500/10 text-gray-600 dark:text-gray-400",
}
