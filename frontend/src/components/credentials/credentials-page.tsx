import { IconLoader2, IconRefresh } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useCredentialsPage } from "@/hooks/use-credentials-page"

import { AnthropicCredentialCard } from "./anthropic-credential-card"
import { AntigravityCredentialCard } from "./antigravity-credential-card"
import { ApiKeyCredentialCard } from "./apikey-credential-card"
import { DeviceCodeSheet } from "./device-code-sheet"
import { LogoutConfirmDialog } from "./logout-confirm-dialog"
import { MiniMaxCredentialCard } from "./minimax-credential-card"
import { OpenAICredentialCard } from "./openai-credential-card"
import { QwenCredentialCard } from "./qwen-credential-card"

// Providers that have dedicated OAuth cards — excluded from API key grid
const OAUTH_PROVIDER_KEYS = new Set([
  "openai",
  "anthropic",
  "antigravity",
  "google-antigravity",
  "qwen",
  "minimax",
])

export function CredentialsPage() {
  const { t } = useTranslation()
  const {
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
    // new
    providerModelGroups,
    apiModelsLoading,
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
  } = useCredentialsPage()

  // Filter out OAuth providers from the API key grid
  const apiKeyGroups = Array.from(providerModelGroups.entries()).filter(
    ([key]) => !OAUTH_PROVIDER_KEYS.has(key),
  )

  // Group image models by provider key for use with ApiKeyCredentialCard
  const imageModelGroups = (() => {
    const groups = new Map<string, typeof imageModels>()
    for (const m of imageModels) {
      const protocol = m.model.split("/")[0].toLowerCase()
      const existing = groups.get(protocol) ?? []
      existing.push(m)
      groups.set(protocol, existing)
    }
    return Array.from(groups.entries())
  })()

  // Group video models by provider key for use with ApiKeyCredentialCard
  const videoModelGroups = (() => {
    const groups = new Map<string, typeof videoModels>()
    for (const m of videoModels) {
      const protocol = m.model.split("/")[0].toLowerCase()
      const existing = groups.get(protocol) ?? []
      existing.push(m)
      groups.set(protocol, existing)
    }
    return Array.from(groups.entries())
  })()

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.credentials")} />

      {error && (
        <div className="text-destructive bg-destructive/10 mx-4 mt-4 rounded-lg px-4 py-3 text-sm sm:mx-6">
          {error}
        </div>
      )}

      {activeFlow && (
        <div className="bg-muted mx-4 mt-4 rounded-lg border px-4 py-3 text-sm sm:mx-6">
          <p className="font-medium">{t("credentials.flow.current")}</p>
          <p className="text-muted-foreground mt-1">{flowHint}</p>
        </div>
      )}

      <Tabs defaultValue="text" className="flex flex-1 flex-col overflow-hidden">
        <TabsList className="mx-4 mt-4 w-fit md:mx-6">
          <TabsTrigger value="text">{t("credentials.tabs.text")}</TabsTrigger>
          <TabsTrigger value="image">{t("credentials.tabs.image")}</TabsTrigger>
          <TabsTrigger value="video">{t("credentials.tabs.video")}</TabsTrigger>
        </TabsList>

        <TabsContent value="text" className="flex-1 overflow-auto">
          <div className="min-h-0 flex-1 px-4 sm:px-6">
            <div className="pt-2">
              <p className="text-muted-foreground text-sm">
                {t("credentials.description")}
              </p>
            </div>

            {loading ? (
              <div className="text-muted-foreground flex items-center gap-2 py-10 text-sm">
                <IconLoader2 className="size-4 animate-spin" />
                {t("credentials.loading")}
              </div>
            ) : (
              <div className="space-y-8 py-5">
                {/* Section 1: OAuth Providers */}
                <section>
                  <h2 className="text-foreground mb-3 text-sm font-semibold">
                    {t("credentials.sections.oauth")}
                  </h2>
                  <div className="grid grid-cols-1 gap-4 lg:auto-rows-fr lg:grid-cols-3 xl:grid-cols-5">
                    <OpenAICredentialCard
                      status={openaiStatus}
                      activeAction={activeAction}
                      token={openAIToken}
                      onTokenChange={setOpenAIToken}
                      onStartBrowserOAuth={() => void startBrowserOAuth("openai")}
                      onStartDeviceCode={() => void startOpenAIDeviceCode()}
                      onStopLoading={stopLoading}
                      onSaveToken={() => void saveToken("openai", openAIToken.trim())}
                      onAskLogout={() => askLogout("openai")}
                    />
                    <AnthropicCredentialCard
                      status={anthropicStatus}
                      activeAction={activeAction}
                      token={anthropicToken}
                      onTokenChange={setAnthropicToken}
                      onStopLoading={stopLoading}
                      onSaveToken={() =>
                        void saveToken("anthropic", anthropicToken.trim())
                      }
                      onStartBrowserOAuth={() =>
                        void startBrowserOAuth("anthropic")
                      }
                      onAskLogout={() => askLogout("anthropic")}
                    />
                    <AntigravityCredentialCard
                      status={antigravityStatus}
                      activeAction={activeAction}
                      onStopLoading={stopLoading}
                      onStartBrowserOAuth={() =>
                        void startBrowserOAuth("google-antigravity")
                      }
                      onAskLogout={() => askLogout("google-antigravity")}
                    />
                    <QwenCredentialCard
                      status={qwenStatus}
                      activeAction={activeAction}
                      onStartDeviceCode={() => void startDeviceCode("qwen")}
                      onStopLoading={stopLoading}
                      onAskLogout={() => askLogout("qwen")}
                    />
                    <MiniMaxCredentialCard
                      status={minimaxStatus}
                      activeAction={activeAction}
                      onStartDeviceCode={() => void startDeviceCode("minimax")}
                      onStopLoading={stopLoading}
                      onAskLogout={() => askLogout("minimax")}
                    />
                  </div>
                </section>

                {/* Section 2: API Key Providers */}
                <section>
                  <div className="mb-3 flex items-center justify-between">
                    <h2 className="text-foreground text-sm font-semibold">
                      {t("credentials.sections.apikey")}
                    </h2>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 gap-1.5 text-xs"
                      onClick={() => void loadApiModels()}
                      disabled={apiModelsLoading}
                    >
                      {apiModelsLoading ? (
                        <IconLoader2 className="size-3 animate-spin" />
                      ) : (
                        <IconRefresh className="size-3" />
                      )}
                      {t("credentials.actions.updateModels")}
                    </Button>
                  </div>

                  {apiModelsLoading && apiKeyGroups.length === 0 ? (
                    <div className="text-muted-foreground flex items-center gap-2 py-6 text-sm">
                      <IconLoader2 className="size-4 animate-spin" />
                      {t("credentials.loading")}
                    </div>
                  ) : (
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                      {apiKeyGroups.map(([provider, models]) => (
                        <ApiKeyCredentialCard
                          key={provider}
                          providerName={provider}
                          models={models}
                          onSaveKey={saveApiKey}
                          onDeleteKey={deleteApiKey}
                          onTestKey={testApiKey}
                        />
                      ))}
                    </div>
                  )}
                </section>
              </div>
            )}
          </div>
        </TabsContent>

        <TabsContent value="image" className="flex-1 overflow-auto p-4 md:p-6">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-foreground text-sm font-semibold">
              {t("credentials.sections.imageModels")}
            </h2>
            <Button
              variant="outline"
              size="sm"
              className="h-7 gap-1.5 text-xs"
              onClick={() => void loadImageModels()}
              disabled={imageModelsLoading}
            >
              {imageModelsLoading ? (
                <IconLoader2 className="size-3 animate-spin" />
              ) : (
                <IconRefresh className="size-3" />
              )}
              {t("credentials.actions.updateModels")}
            </Button>
          </div>

          {imageModelsLoading && imageModelGroups.length === 0 ? (
            <div className="text-muted-foreground flex items-center gap-2 py-6 text-sm">
              <IconLoader2 className="size-4 animate-spin" />
              {t("credentials.loading")}
            </div>
          ) : imageModelGroups.length === 0 ? (
            <p className="text-muted-foreground py-6 text-sm">
              {t("credentials.noModels")}
            </p>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {imageModelGroups.map(([provider, models]) => (
                <ApiKeyCredentialCard
                  key={provider}
                  providerName={provider}
                  models={models}
                  onSaveKey={saveImageApiKey}
                  onDeleteKey={deleteImageApiKey}
                  onTestKey={testImageApiKey}
                />
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent value="video" className="flex-1 overflow-auto p-4 md:p-6">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-foreground text-sm font-semibold">
              {t("credentials.sections.videoModels")}
            </h2>
            <Button
              variant="outline"
              size="sm"
              className="h-7 gap-1.5 text-xs"
              onClick={() => void loadVideoModels()}
              disabled={videoModelsLoading}
            >
              {videoModelsLoading ? (
                <IconLoader2 className="size-3 animate-spin" />
              ) : (
                <IconRefresh className="size-3" />
              )}
              {t("credentials.actions.updateModels")}
            </Button>
          </div>

          {videoModelsLoading && videoModelGroups.length === 0 ? (
            <div className="text-muted-foreground flex items-center gap-2 py-6 text-sm">
              <IconLoader2 className="size-4 animate-spin" />
              {t("credentials.loading")}
            </div>
          ) : videoModelGroups.length === 0 ? (
            <p className="text-muted-foreground py-6 text-sm">
              {t("credentials.noModels")}
            </p>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {videoModelGroups.map(([provider, models]) => (
                <ApiKeyCredentialCard
                  key={provider}
                  providerName={provider}
                  models={models}
                  onSaveKey={saveVideoApiKey}
                  onDeleteKey={deleteVideoApiKey}
                  onTestKey={testVideoApiKey}
                />
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>

      <LogoutConfirmDialog
        open={logoutDialogOpen}
        providerLabel={logoutProviderLabel}
        isSubmitting={activeAction === `${logoutConfirmProvider}:logout`}
        onOpenChange={handleLogoutDialogOpenChange}
        onConfirm={handleConfirmLogout}
      />

      <DeviceCodeSheet
        open={deviceSheetOpen}
        flow={deviceFlow}
        flowHint={flowHint}
        onOpenChange={handleDeviceSheetOpenChange}
      />
    </div>
  )
}
