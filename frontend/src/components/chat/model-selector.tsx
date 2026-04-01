import { IconSparkles } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { ModelInfo } from "@/api/models"
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

interface ModelSelectorProps {
  defaultModelName: string
  apiKeyModels: ModelInfo[]
  oauthModels: ModelInfo[]
  localModels: ModelInfo[]
  onValueChange: (modelName: string) => void
  isAutoMode: boolean
  toggleAutoMode: (enabled: boolean) => void
}

export function ModelSelector({
  defaultModelName,
  apiKeyModels,
  oauthModels,
  localModels,
  onValueChange,
  isAutoMode,
  toggleAutoMode,
}: ModelSelectorProps) {
  const { t } = useTranslation()

  function handleValueChange(value: string) {
    if (value === "__auto__") {
      void toggleAutoMode(true)
    } else {
      if (isAutoMode) {
        void toggleAutoMode(false)
      }
      onValueChange(value)
    }
  }

  return (
    <Select
      value={isAutoMode ? "__auto__" : defaultModelName}
      onValueChange={handleValueChange}
    >
      <SelectTrigger
        size="sm"
        className="text-muted-foreground hover:text-foreground focus-visible:border-input h-8 max-w-[160px] min-w-[80px] bg-transparent shadow-none focus-visible:ring-0 sm:max-w-[220px]"
      >
        <SelectValue placeholder={t("chat.noModel")} />
      </SelectTrigger>
      <SelectContent position="popper" align="start">
        <SelectGroup>
          <SelectItem value="__auto__">
            <div className="flex items-center gap-2">
              <IconSparkles className="size-4 text-violet-400" />
              <span>Auto</span>
              {isAutoMode && (
                <span className="text-xs text-muted-foreground ml-1">
                  (active)
                </span>
              )}
            </div>
          </SelectItem>
        </SelectGroup>

        {(apiKeyModels.length > 0 ||
          oauthModels.length > 0 ||
          localModels.length > 0) && <SelectSeparator />}

        {apiKeyModels.length > 0 && (
          <SelectGroup>
            <SelectLabel>{t("chat.modelGroup.apikey")}</SelectLabel>
            {apiKeyModels.map((model) => (
              <SelectItem key={model.index} value={model.model_name}>
                {model.model_name}
              </SelectItem>
            ))}
          </SelectGroup>
        )}
        {apiKeyModels.length > 0 &&
          (oauthModels.length > 0 || localModels.length > 0) && (
            <SelectSeparator />
          )}

        {oauthModels.length > 0 && (
          <SelectGroup>
            <SelectLabel>{t("chat.modelGroup.oauth")}</SelectLabel>
            {oauthModels.map((model) => (
              <SelectItem key={model.index} value={model.model_name}>
                {model.model_name}
              </SelectItem>
            ))}
          </SelectGroup>
        )}
        {oauthModels.length > 0 &&
          (localModels.length > 0 || apiKeyModels.length > 0) && (
            <SelectSeparator />
          )}

        {localModels.length > 0 && (
          <SelectGroup>
            <SelectLabel>{t("chat.modelGroup.local")}</SelectLabel>
            {localModels.map((model) => (
              <SelectItem key={model.index} value={model.model_name}>
                {model.model_name}
              </SelectItem>
            ))}
          </SelectGroup>
        )}
      </SelectContent>
    </Select>
  )
}
