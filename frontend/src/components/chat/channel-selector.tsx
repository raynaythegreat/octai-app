import {
  IconBrandChrome,
  IconBrandDingtalk,
  IconBrandDiscord,
  IconBrandLine,
  IconBrandMatrix,
  IconBrandQq,
  IconBrandSlack,
  IconBrandTelegram,
  IconBrandWechat,
  IconBrandWhatsapp,
  IconMessages,
  IconTerminal2,
} from "@tabler/icons-react"
import * as React from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  getAppConfig,
  getChannelsCatalog,
  type SupportedChannel,
} from "@/api/channels"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const CHANNEL_ICONS: Record<
  string,
  React.ComponentType<{ className?: string }>
> = {
  telegram: IconBrandTelegram,
  discord: IconBrandDiscord,
  slack: IconBrandSlack,
  feishu: IconBrandSlack,
  dingtalk: IconBrandDingtalk,
  line: IconBrandLine,
  qq: IconBrandQq,
  weixin: IconBrandWechat,
  wecom: IconBrandWechat,
  whatsapp: IconBrandWhatsapp,
  whatsapp_native: IconBrandWhatsapp,
  matrix: IconBrandMatrix,
  irc: IconMessages,
}

function getDisplayName(name: string): string {
  const names: Record<string, string> = {
    telegram: "Telegram",
    discord: "Discord",
    slack: "Slack",
    feishu: "Feishu / Lark",
    dingtalk: "DingTalk",
    line: "LINE",
    qq: "QQ",
    weixin: "WeChat",
    wecom: "WeCom",
    whatsapp: "WhatsApp",
    whatsapp_native: "WhatsApp (Native)",
    matrix: "Matrix",
    irc: "IRC",
    maixcam: "MaixCAM",
    onebot: "OneBot",
  }
  return names[name] ?? name.charAt(0).toUpperCase() + name.slice(1)
}

interface ChannelEntry {
  channel: SupportedChannel
  enabled: boolean
}

interface ChannelSelectorProps {
  activeChannel?: string
  onChannelChange?: (channel: string) => void
}

export function ChannelSelector({ activeChannel, onChannelChange }: ChannelSelectorProps) {
  const { t } = useTranslation()
  const [entries, setEntries] = React.useState<ChannelEntry[]>([])

  React.useEffect(() => {
    let active = true
    Promise.all([getChannelsCatalog(), getAppConfig().catch(() => ({}))])
      .then(([catalog, config]) => {
        if (!active) return
        const channelsConfig =
          ((config as Record<string, unknown>).channels as Record<
            string,
            unknown
          >) ?? {}
        const list: ChannelEntry[] = catalog.channels
          .filter((ch) => ch.name !== "pico" && ch.name !== "maixcam")
          .map((ch) => {
            const chCfg = channelsConfig[ch.config_key] as
              | Record<string, unknown>
              | undefined
            let enabled = chCfg?.enabled === true
            if (enabled && ch.name === "whatsapp") {
              enabled = chCfg?.use_native !== true
            }
            if (enabled && ch.name === "whatsapp_native") {
              enabled = chCfg?.use_native === true
            }
            return { channel: ch, enabled }
          })
        setEntries(list)
      })
      .catch(() => {})
    return () => {
      active = false
    }
  }, [])

  const enabledChannels = entries.filter((e) => e.enabled)
  const disabledChannels = entries.filter((e) => !e.enabled)

  const selectValue =
    activeChannel === "pico" || !activeChannel
      ? "web"
      : activeChannel === "cli"
        ? "tui"
        : activeChannel

  return (
    <Select
      value={selectValue}
      onValueChange={(val) => {
        if (val !== "web" && val !== "tui" && onChannelChange) {
          const entry = entries.find((e) => e.channel.name === val)
          if (!entry?.enabled) {
            toast.info(t("chat.channelSwitchHint"))
            return
          }
        }
        if (onChannelChange) {
          const channelKey = val === "web" ? "pico" : val === "tui" ? "cli" : val
          onChannelChange(channelKey)
        }
      }}
    >
      <SelectTrigger className="h-8 w-auto gap-1.5 border-0 bg-transparent px-2 text-sm shadow-none focus:ring-0">
        <IconBrandChrome className="size-4 opacity-70" />
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          <SelectLabel className="text-xs">{t("chat.channel.active")}</SelectLabel>
          <SelectItem value="web">
            <div className="flex items-center gap-2">
              <IconBrandChrome className="size-4" />
              {t("chat.channel.web")}
            </div>
          </SelectItem>
          <SelectItem value="tui">
            <div className="flex items-center gap-2">
              <IconTerminal2 className="size-4" />
              {t("chat.channel.tui")}
            </div>
          </SelectItem>
          {enabledChannels.map(({ channel }) => {
            const Icon = CHANNEL_ICONS[channel.name] ?? IconMessages
            return (
              <SelectItem key={channel.name} value={channel.name}>
                <div className="flex items-center gap-2">
                  <Icon className="size-4" />
                  {getDisplayName(channel.name)}
                </div>
              </SelectItem>
            )
          })}
        </SelectGroup>

        {disabledChannels.length > 0 && (
          <>
            <SelectSeparator />
            <SelectGroup>
              <SelectLabel className="text-xs">{t("chat.channel.notConfigured")}</SelectLabel>
              {disabledChannels.map(({ channel }) => {
                const Icon = CHANNEL_ICONS[channel.name] ?? IconMessages
                return (
                  <SelectItem
                    key={channel.name}
                    value={channel.name}
                    className="text-muted-foreground"
                  >
                    <div className="flex items-center gap-2">
                      <Icon className="size-4 opacity-50" />
                      {getDisplayName(channel.name)}
                    </div>
                  </SelectItem>
                )
              })}
            </SelectGroup>
          </>
        )}
      </SelectContent>
    </Select>
  )
}
