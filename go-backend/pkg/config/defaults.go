// OctAi - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OctAi contributors

package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/raynaythegreat/octai-app/pkg"
)

// DefaultConfig returns the default configuration for OctAi.
func DefaultConfig() *Config {
	// Determine the base path for the workspace.
	// Priority: $OCTAI_HOME > ~/.octai
	var homePath string
	if octaiHome := os.Getenv(EnvHome); octaiHome != "" {
		homePath = octaiHome
	} else {
		userHome, _ := os.UserHomeDir()
		homePath = filepath.Join(userHome, pkg.DefaultAIBusinessHQHome)
	}
	workspacePath := filepath.Join(homePath, pkg.WorkspaceName)

	return &Config{
		Version: CurrentVersion,
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace:                 workspacePath,
				RestrictToWorkspace:       true,
				Provider:                  "",
				MaxTokens:                 32768,
				Temperature:               nil, // nil means use provider default
				MaxToolIterations:         50,
				SummarizeMessageThreshold: 20,
				SummarizeTokenPercent:     75,
				SteeringMode:              "one-at-a-time",
				ToolFeedback: ToolFeedbackConfig{
					Enabled:       true,
					MaxArgsLength: 300,
				},
			},
		},
		Bindings: []AgentBinding{},
		Session: SessionConfig{
			DMScope: "per-channel-peer",
		},
		Channels: ChannelsConfig{
			WhatsApp: WhatsAppConfig{
				Enabled:          false,
				BridgeURL:        "ws://localhost:3001",
				UseNative:        false,
				SessionStorePath: "",
				AllowFrom:        FlexibleStringSlice{},
			},
			Telegram: TelegramConfig{
				Enabled:   false,
				AllowFrom: FlexibleStringSlice{},
				Typing:    TypingConfig{Enabled: true},
				Placeholder: PlaceholderConfig{
					Enabled: true,
					Text:    "Thinking... 💭",
				},
				Streaming:     StreamingConfig{Enabled: true, ThrottleSeconds: 3, MinGrowthChars: 200},
				UseMarkdownV2: false,
			},
			Feishu: FeishuConfig{
				Enabled:   false,
				AppID:     "",
				AllowFrom: FlexibleStringSlice{},
			},
			Discord: DiscordConfig{
				Enabled:     false,
				AllowFrom:   FlexibleStringSlice{},
				MentionOnly: false,
			},
			MaixCam: MaixCamConfig{
				Enabled:   false,
				Host:      "0.0.0.0",
				Port:      18790,
				AllowFrom: FlexibleStringSlice{},
			},
			QQ: QQConfig{
				Enabled:              false,
				AppID:                "",
				AllowFrom:            FlexibleStringSlice{},
				MaxMessageLength:     2000,
				MaxBase64FileSizeMiB: 0,
			},
			DingTalk: DingTalkConfig{
				Enabled:   false,
				ClientID:  "",
				AllowFrom: FlexibleStringSlice{},
			},
			Slack: SlackConfig{
				Enabled:   false,
				AllowFrom: FlexibleStringSlice{},
			},
			Matrix: MatrixConfig{
				Enabled:      false,
				Homeserver:   "https://matrix.org",
				UserID:       "",
				DeviceID:     "",
				JoinOnInvite: true,
				AllowFrom:    FlexibleStringSlice{},
				GroupTrigger: GroupTriggerConfig{
					MentionOnly: true,
				},
				Placeholder: PlaceholderConfig{
					Enabled: true,
					Text:    "Thinking... 💭",
				},
				CryptoDatabasePath: "",
				CryptoPassphrase:   "",
			},
			LINE: LINEConfig{
				Enabled:      false,
				WebhookHost:  "0.0.0.0",
				WebhookPort:  18791,
				WebhookPath:  "/webhook/line",
				AllowFrom:    FlexibleStringSlice{},
				GroupTrigger: GroupTriggerConfig{MentionOnly: true},
			},
			OneBot: OneBotConfig{
				Enabled:           false,
				WSUrl:             "ws://127.0.0.1:3001",
				ReconnectInterval: 5,
				AllowFrom:         FlexibleStringSlice{},
			},
			WeCom: WeComConfig{
				Enabled:             false,
				BotID:               "",
				WebSocketURL:        "wss://openws.work.weixin.qq.com",
				SendThinkingMessage: true,
				AllowFrom:           FlexibleStringSlice{},
			},
			Weixin: WeixinConfig{
				Enabled:    false,
				BaseURL:    "https://ilinkai.weixin.qq.com/",
				CDNBaseURL: "https://novac2c.cdn.weixin.qq.com/c2c",
				AllowFrom:  FlexibleStringSlice{},
				Proxy:      "",
			},
			Pico: PicoConfig{
				Enabled:        false,
				PingInterval:   30,
				ReadTimeout:    60,
				WriteTimeout:   10,
				MaxConnections: 100,
				AllowFrom:      FlexibleStringSlice{},
			},
		},
		Hooks: HooksConfig{
			Enabled: true,
			Defaults: HookDefaultsConfig{
				ObserverTimeoutMS:    500,
				InterceptorTimeoutMS: 5000,
				ApprovalTimeoutMS:    60000,
			},
		},
		ModelList: []*ModelConfig{
			// ============================================
			// Add your API key to the model you want to use
			// ============================================

			// Zhipu AI (智谱) - https://open.bigmodel.cn/usercenter/apikeys
			{
				ModelName: "glm-4.7",
				Model:     "zhipu/glm-4.7",
				APIBase:   "https://open.bigmodel.cn/api/paas/v4",
			},

			// OpenAI - https://platform.openai.com/api-keys
			{
				ModelName: "gpt-5.4",
				Model:     "openai/gpt-5.4",
				APIBase:   "https://api.openai.com/v1",
			},
			{
				ModelName: "gpt-5",
				Model:     "openai/gpt-5",
				APIBase:   "https://api.openai.com/v1",
			},
			{
				ModelName: "gpt-5.4-mini",
				Model:     "openai/gpt-5.4-mini",
				APIBase:   "https://api.openai.com/v1",
			},
			{
				ModelName: "gpt-5.4-nano",
				Model:     "openai/gpt-5.4-nano",
				APIBase:   "https://api.openai.com/v1",
			},

			// Anthropic Claude - https://console.anthropic.com/settings/keys
			{
				ModelName: "claude-sonnet-4.6",
				Model:     "anthropic/claude-sonnet-4.6",
				APIBase:   "https://api.anthropic.com/v1",
			},

			// DeepSeek - https://platform.deepseek.com/
			{
				ModelName: "deepseek-chat",
				Model:     "deepseek/deepseek-chat",
				APIBase:   "https://api.deepseek.com/v1",
			},

			// Google Gemini - https://ai.google.dev/

			// Qwen (通义千问) - https://dashscope.console.aliyun.com/apiKey
			{
				ModelName: "qwen-plus",
				Model:     "qwen/qwen-plus",
				APIBase:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
			},

			// Moonshot (月之暗面) - https://platform.moonshot.cn/console/api-keys

			// Groq - https://console.groq.com/keys
			{
				ModelName: "llama-3.3-70b-versatile",
				Model:     "groq/llama-3.3-70b-versatile",
				APIBase:   "https://api.groq.com/openai/v1",
			},

			// OpenRouter (100+ models) - https://openrouter.ai/keys
			{
				ModelName: "openrouter-auto",
				Model:     "openrouter/auto",
				APIBase:   "https://openrouter.ai/api/v1",
			},
			{
				ModelName: "openrouter-gpt-5.4",
				Model:     "openrouter/openai/gpt-5.4",
				APIBase:   "https://openrouter.ai/api/v1",
			},

			// NVIDIA - https://build.nvidia.com/
			{
				ModelName: "nemotron-4-340b",
				Model:     "nvidia/nemotron-4-340b-instruct",
				APIBase:   "https://integrate.api.nvidia.com/v1",
			},

			// Cerebras - https://inference.cerebras.ai/

			// Vivgrid - https://vivgrid.com
			{
				ModelName: "vivgrid-auto",
				Model:     "vivgrid/auto",
				APIBase:   "https://api.vivgrid.com/v1",
			},

			// Volcengine (火山引擎) - https://console.volcengine.com/ark
			{
				ModelName: "ark-code-latest",
				Model:     "volcengine/ark-code-latest",
				APIBase:   "https://ark.cn-beijing.volces.com/api/v3",
			},
			{
				ModelName: "doubao-pro",
				Model:     "volcengine/doubao-pro-32k",
				APIBase:   "https://ark.cn-beijing.volces.com/api/v3",
			},

			// ShengsuanYun (神算云)

			// Antigravity (Google Cloud Code Assist) - OAuth only
			{
				ModelName:  "gemini-flash",
				Model:      "antigravity/gemini-3-flash",
				AuthMethod: "oauth",
			},

			// GitHub Copilot - https://github.com/settings/tokens
			{
				ModelName:  "copilot-gpt-5.4",
				Model:      "github-copilot/gpt-5.4",
				APIBase:    "http://localhost:4321",
				AuthMethod: "oauth",
			},

			// Ollama (local) - https://ollama.com
			{
				ModelName: "llama3",
				Model:     "ollama/llama3",
				APIBase:   "http://localhost:11434/v1",
			},

			// Mistral AI - https://console.mistral.ai/api-keys
			{
				ModelName: "mistral-small",
				Model:     "mistral/mistral-small-latest",
				APIBase:   "https://api.mistral.ai/v1",
			},

			// Avian - https://avian.io
			{
				ModelName: "deepseek-v3.2",
				Model:     "avian/deepseek/deepseek-v3.2",
				APIBase:   "https://api.avian.io/v1",
			},
			{
				ModelName: "kimi-k2.5",
				Model:     "avian/moonshotai/kimi-k2.5",
				APIBase:   "https://api.avian.io/v1",
			},

			// Minimax - https://api.minimaxi.com/
			{
				ModelName: "MiniMax-M2.5",
				Model:     "minimax/MiniMax-M2.5",
				APIBase:   "https://api.minimaxi.com/v1",
				ExtraBody: map[string]any{"reasoning_split": true},
			},

			// LongCat - https://longcat.chat/platform
			{
				ModelName: "LongCat-Flash-Thinking",
				Model:     "longcat/LongCat-Flash-Thinking",
				APIBase:   "https://api.longcat.chat/openai",
			},

			// ModelScope (魔搭社区) - https://modelscope.cn/my/tokens
			{
				ModelName: "modelscope-qwen",
				Model:     "modelscope/Qwen/Qwen3-235B-A22B-Instruct-2507",
				APIBase:   "https://api-inference.modelscope.cn/v1",
			},

			// VLLM (local) - http://localhost:8000
			{
				ModelName: "local-model",
				Model:     "vllm/custom-model",
				APIBase:   "http://localhost:8000/v1",
			},

			// Azure OpenAI - https://portal.azure.com
			// model_name is a user-friendly alias; the model field's path after "azure/" is your deployment name
			{
				ModelName: "azure-gpt5",
				Model:     "azure/my-gpt5-deployment",
				APIBase:   "https://your-resource.openai.azure.com",
			},

			// Anthropic family
			{ModelName: "claude-opus-4-6", Model: "anthropic/claude-opus-4-6", APIBase: "https://api.anthropic.com/v1"},
			{ModelName: "claude-haiku-4-5", Model: "anthropic/claude-haiku-4-5-20251001", APIBase: "https://api.anthropic.com/v1"},
			{ModelName: "claude-opus-4-5", Model: "anthropic/claude-opus-4-5", APIBase: "https://api.anthropic.com/v1"},
			{ModelName: "claude-sonnet-4-5", Model: "anthropic/claude-sonnet-4-5", APIBase: "https://api.anthropic.com/v1"},

			// OpenAI family
			{ModelName: "o3", Model: "openai/o3", APIBase: "https://api.openai.com/v1"},
			{ModelName: "o4-mini", Model: "openai/o4-mini", APIBase: "https://api.openai.com/v1"},

			// Groq family
			{ModelName: "llama-3.1-8b-instant", Model: "groq/llama-3.1-8b-instant", APIBase: "https://api.groq.com/openai/v1"},
			{ModelName: "llama-4-scout-17b", Model: "groq/meta-llama/llama-4-scout-17b-16e-instruct", APIBase: "https://api.groq.com/openai/v1"},
			{ModelName: "qwen3-32b", Model: "groq/qwen/qwen3-32b", APIBase: "https://api.groq.com/openai/v1"},

			// Cerebras
			{ModelName: "cerebras-gpt-oss-120b", Model: "cerebras/gpt-oss-120b", APIBase: "https://api.cerebras.ai/v1"},
			{ModelName: "cerebras-llama-3.1-8b", Model: "cerebras/llama3.1-8b", APIBase: "https://api.cerebras.ai/v1"},

			// DeepSeek
			{ModelName: "deepseek-reasoner", Model: "deepseek/deepseek-reasoner", APIBase: "https://api.deepseek.com/v1"},

			// xAI Grok
			{ModelName: "grok-4-reasoning", Model: "grok/grok-4.20-0309-reasoning", APIBase: "https://api.x.ai/v1"},
			{ModelName: "grok-4-non-reasoning", Model: "grok/grok-4.20-0309-non-reasoning", APIBase: "https://api.x.ai/v1"},
			{ModelName: "grok-4-fast", Model: "grok/grok-4-1-fast-reasoning", APIBase: "https://api.x.ai/v1"},

			// Google Gemini (more models)
			{ModelName: "gemini-2.5-pro", Model: "gemini/gemini-2.5-pro", APIBase: "https://generativelanguage.googleapis.com/v1beta"},
			{ModelName: "gemini-2.5-flash", Model: "gemini/gemini-2.5-flash", APIBase: "https://generativelanguage.googleapis.com/v1beta"},
			{ModelName: "gemini-2.5-flash-lite", Model: "gemini/gemini-2.5-flash-lite", APIBase: "https://generativelanguage.googleapis.com/v1beta"},
			{ModelName: "gemini-3-flash-preview", Model: "gemini/gemini-3-flash-preview", APIBase: "https://generativelanguage.googleapis.com/v1beta"},

			// Mistral AI (more models)
			{ModelName: "mistral-large", Model: "mistral/mistral-large-latest", APIBase: "https://api.mistral.ai/v1"},
			{ModelName: "mistral-medium", Model: "mistral/mistral-medium-latest", APIBase: "https://api.mistral.ai/v1"},
			{ModelName: "codestral", Model: "mistral/codestral-latest", APIBase: "https://api.mistral.ai/v1"},
			{ModelName: "devstral-small", Model: "mistral/devstral-small-2-25-12", APIBase: "https://api.mistral.ai/v1"},

			// Moonshot / Kimi
			{ModelName: "kimi-k2-thinking", Model: "moonshot/kimi-k2-thinking", APIBase: "https://api.moonshot.cn/v1"},
			{ModelName: "kimi-k2-turbo", Model: "moonshot/kimi-k2-turbo-preview", APIBase: "https://api.moonshot.cn/v1"},

			// Together AI
			{ModelName: "together-llama-4-maverick", Model: "together/meta-llama/Llama-4-Maverick-17B-128E-Instruct", APIBase: "https://api.together.xyz/v1"},
			{ModelName: "together-qwen3", Model: "together/Qwen/Qwen3-235B-A22B-Instruct-2507-FP8", APIBase: "https://api.together.xyz/v1"},
			{ModelName: "together-deepseek-r1", Model: "together/deepseek-ai/DeepSeek-R1", APIBase: "https://api.together.xyz/v1"},
			{ModelName: "together-kimi-k2", Model: "together/moonshotai/Kimi-K2-Instruct", APIBase: "https://api.together.xyz/v1"},

			// Perplexity (web-search augmented)
			{ModelName: "sonar", Model: "perplexity/sonar", APIBase: "https://api.perplexity.ai"},
			{ModelName: "sonar-pro", Model: "perplexity/sonar-pro", APIBase: "https://api.perplexity.ai"},
			{ModelName: "sonar-reasoning", Model: "perplexity/sonar-reasoning", APIBase: "https://api.perplexity.ai"},
			{ModelName: "sonar-reasoning-pro", Model: "perplexity/sonar-reasoning-pro", APIBase: "https://api.perplexity.ai"},

			// Ollama (more common models)
			{ModelName: "llama4", Model: "ollama/llama4", APIBase: "http://localhost:11434/v1"},
			{ModelName: "llama3.2", Model: "ollama/llama3.2", APIBase: "http://localhost:11434/v1"},
			{ModelName: "llama3.1", Model: "ollama/llama3.1", APIBase: "http://localhost:11434/v1"},
			{ModelName: "qwen3", Model: "ollama/qwen3", APIBase: "http://localhost:11434/v1"},
			{ModelName: "qwen2.5", Model: "ollama/qwen2.5", APIBase: "http://localhost:11434/v1"},
			{ModelName: "mistral", Model: "ollama/mistral", APIBase: "http://localhost:11434/v1"},
			{ModelName: "codellama", Model: "ollama/codellama", APIBase: "http://localhost:11434/v1"},
			{ModelName: "phi3", Model: "ollama/phi3", APIBase: "http://localhost:11434/v1"},
			{ModelName: "gemma2", Model: "ollama/gemma2", APIBase: "http://localhost:11434/v1"},

			// OpenRouter (more specific models)
			{ModelName: "openrouter-claude-sonnet", Model: "openrouter/anthropic/claude-sonnet-4-6", APIBase: "https://openrouter.ai/api/v1"},
			{ModelName: "openrouter-deepseek-v3", Model: "openrouter/deepseek/deepseek-chat", APIBase: "https://openrouter.ai/api/v1"},

			// LiteLLM proxy
			{ModelName: "litellm-auto", Model: "litellm/auto", APIBase: "http://localhost:4000/v1"},
		},
		ImageModelList: []*ModelConfig{
			// OpenAI DALL-E - https://platform.openai.com/api-keys
			{
				ModelName: "dall-e-3",
				Model:     "openai/dall-e-3",
				APIBase:   "https://api.openai.com/v1",
			},
			// Google Imagen - https://ai.google.dev/
			{
				ModelName: "gemini-imagen-4",
				Model:     "google/imagen-4",
				APIBase:   "https://generativelanguage.googleapis.com/v1beta",
			},
			// Stability AI - https://platform.stability.ai/
			{
				ModelName: "stability-sdxl",
				Model:     "stability/stable-diffusion-xl-1024-v1-0",
				APIBase:   "https://api.stability.ai/v1",
			},
			// Together AI / Black Forest Labs FLUX - https://api.together.xyz/
			{
				ModelName: "flux-schnell",
				Model:     "together/black-forest-labs/FLUX.1-schnell",
				APIBase:   "https://api.together.xyz/v1",
			},
		},
		VideoModelList: []*ModelConfig{
			// Runway Gen-4 - https://runwayml.com/
			{
				ModelName: "runway-gen4",
				Model:     "runway/gen4-turbo",
				APIBase:   "https://api.runwayml.com/v1",
			},
			// Kling AI - https://klingai.com/
			{
				ModelName: "kling-v2",
				Model:     "kling/v2-master",
				APIBase:   "https://api.klingai.com/v1",
			},
			// Google Veo - https://ai.google.dev/
			{
				ModelName: "google-veo-2",
				Model:     "google/veo-2",
				APIBase:   "https://generativelanguage.googleapis.com/v1beta",
			},
			// MiniMax Video - https://api.minimax.chat/
			{
				ModelName: "minimax-video",
				Model:     "minimax/video-01",
				APIBase:   "https://api.minimax.chat/v1",
			},
		},
		Gateway: GatewayConfig{
			Host:      "127.0.0.1",
			Port:      18790,
			HotReload: false,
			LogLevel:  "fatal",
		},
		Tools: ToolsConfig{
			FilterSensitiveData: true,
			FilterMinLength:     8,
			MediaCleanup: MediaCleanupConfig{
				ToolConfig: ToolConfig{
					Enabled: true,
				},
				MaxAge:   30,
				Interval: 5,
			},
			Web: WebToolsConfig{
				ToolConfig: ToolConfig{
					Enabled: true,
				},
				PreferNative:    true,
				Proxy:           "",
				FetchLimitBytes: 10 * 1024 * 1024, // 10MB by default
				Format:          "plaintext",
				Brave: BraveConfig{
					Enabled:    false,
					MaxResults: 5,
				},
				Tavily: TavilyConfig{
					Enabled:    false,
					MaxResults: 5,
				},
				DuckDuckGo: DuckDuckGoConfig{
					Enabled:    true,
					MaxResults: 5,
				},
				Perplexity: PerplexityConfig{
					Enabled:    false,
					MaxResults: 5,
				},
				SearXNG: SearXNGConfig{
					Enabled:    false,
					BaseURL:    "",
					MaxResults: 5,
				},
				GLMSearch: GLMSearchConfig{
					Enabled:      false,
					BaseURL:      "https://open.bigmodel.cn/api/paas/v4/web_search",
					SearchEngine: "search_std",
					MaxResults:   5,
				},
				BaiduSearch: BaiduSearchConfig{
					Enabled:    false,
					BaseURL:    "https://qianfan.baidubce.com/v2/ai_search/web_search",
					MaxResults: 10,
				},
			},
			Cron: CronToolsConfig{
				ToolConfig: ToolConfig{
					Enabled: true,
				},
				ExecTimeoutMinutes: 5,
				AllowCommand:       true,
			},
			Exec: ExecConfig{
				ToolConfig: ToolConfig{
					Enabled: true,
				},
				EnableDenyPatterns: true,
				AllowRemote:        true,
				TimeoutSeconds:     60,
			},
			Skills: SkillsToolsConfig{
				ToolConfig: ToolConfig{
					Enabled: true,
				},
				Registries: SkillsRegistriesConfig{
					ClawHub: ClawHubRegistryConfig{
						Enabled: true,
						BaseURL: "https://clawhub.ai",
					},
				},
				MaxConcurrentSearches: 2,
				SearchCache: SearchCacheConfig{
					MaxSize:    50,
					TTLSeconds: 300,
				},
			},
			SendFile: ToolConfig{
				Enabled: true,
			},
			MCP: MCPConfig{
				ToolConfig: ToolConfig{
					Enabled: false,
				},
				Discovery: ToolDiscoveryConfig{
					Enabled:          false,
					TTL:              5,
					MaxSearchResults: 5,
					UseBM25:          true,
					UseRegex:         false,
				},
				Servers: map[string]MCPServerConfig{
					"notebooklm": {
						Enabled: false,
						Command: "uvx",
						Args:    []string{"--from", "notebooklm-mcp-cli", "notebooklm-mcp"},
					},
					"filesystem": {
						Enabled:  false,
						Deferred: boolPtr(true),
						Command:  "npx",
						Args:     []string{"-y", "@anthropic/mcp-server-filesystem", "<workspace>"},
					},
					"sequential-thinking": {
						Enabled:  false,
						Deferred: boolPtr(true),
						Command:  "npx",
						Args:     []string{"-y", "@anthropic/mcp-server-sequential-thinking"},
					},
					"memory": {
						Enabled:  false,
						Deferred: boolPtr(true),
						Command:  "npx",
						Args:     []string{"-y", "@anthropic/mcp-server-memory"},
					},
					"github": {
						Enabled:  false,
						Deferred: boolPtr(true),
						Command:  "npx",
						Args:     []string{"-y", "@anthropic/mcp-server-github"},
						EnvFile:  ".env.github",
					},
					"brave-search": {
						Enabled:  false,
						Deferred: boolPtr(true),
						Command:  "npx",
						Args:     []string{"-y", "@anthropic/mcp-server-brave-search"},
						EnvFile:  ".env.brave",
					},
					"puppeteer": {
						Enabled:  false,
						Deferred: boolPtr(true),
						Command:  "npx",
						Args:     []string{"-y", "@anthropic/mcp-server-puppeteer"},
					},
					"fetch": {
						Enabled:  false,
						Deferred: boolPtr(true),
						Command:  "npx",
						Args:     []string{"-y", "@anthropic/mcp-server-fetch"},
					},
				},
			},
			AppendFile: ToolConfig{
				Enabled: true,
			},
			EditFile: ToolConfig{
				Enabled: true,
			},
			FindSkills: ToolConfig{
				Enabled: true,
			},
			I2C: ToolConfig{
				Enabled: false, // Hardware tool - Linux only
			},
			InstallSkill: ToolConfig{
				Enabled: true,
			},
			ListDir: ToolConfig{
				Enabled: true,
			},
			Message: ToolConfig{
				Enabled: true,
			},
			ReadFile: ReadFileToolConfig{
				Enabled:         true,
				MaxReadFileSize: 64 * 1024, // 64KB
			},
			Spawn: ToolConfig{
				Enabled: true,
			},
			SpawnStatus: ToolConfig{
				Enabled: false,
			},
			SPI: ToolConfig{
				Enabled: false, // Hardware tool - Linux only
			},
			Subagent: ToolConfig{
				Enabled: true,
			},
			WebFetch: ToolConfig{
				Enabled: true,
			},
			WriteFile: ToolConfig{
				Enabled: true,
			},
		},
		Heartbeat: HeartbeatConfig{
			Enabled:  true,
			Interval: 30,
		},
		Devices: DevicesConfig{
			Enabled:    false,
			MonitorUSB: true,
		},
		Voice: VoiceConfig{
			ModelName:         "",
			EchoTranscription: false,
		},
		BuildInfo: BuildInfo{
			Version:   Version,
			GitCommit: GitCommit,
			BuildTime: BuildTime,
			GoVersion: GoVersion,
		},
		security: &SecurityConfig{
			ModelList: map[string]ModelSecurityEntry{},
			Channels:  &ChannelsSecurity{},
			Web:       &WebToolsSecurity{},
			Skills:    &SkillsSecurity{},
		},
	}
}

// ModelNamesForProvider returns the ordered list of model names from DefaultModelConfigs
// whose Model field starts with the given schemePrefix (e.g. "openai", "gemini").
var modelNamesCache sync.Map

func ModelNamesForProvider(schemePrefix string) []string {
	if cached, ok := modelNamesCache.Load(schemePrefix); ok {
		return cached.([]string)
	}

	var names []string
	prefix := schemePrefix + "/"
	for _, m := range DefaultConfig().ModelList {
		if strings.HasPrefix(m.Model, prefix) {
			names = append(names, m.ModelName)
		}
	}

	result := make([]string, len(names))
	copy(result, names)
	modelNamesCache.Store(schemePrefix, result)
	return result
}

func boolPtr(v bool) *bool {
	return &v
}
