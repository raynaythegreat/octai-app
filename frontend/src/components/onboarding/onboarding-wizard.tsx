import { useTranslation } from "react-i18next"
import { useCallback, useEffect, useRef, useState } from "react"
import { IconCheck, IconChevronLeft, IconChevronRight, IconDeviceMobile, IconLanguage, IconPlug, IconRocket, IconServer } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"

import "./onboarding.css"

const ONBOARDING_KEY = "octai-onboarded"

const TOTAL_STEPS = 5

interface Provider {
  id: string
  name: string
  icon: string
}

const PROVIDERS: Provider[] = [
  { id: "openai", name: "OpenAI", icon: "🤖" },
  { id: "anthropic", name: "Anthropic", icon: "🧠" },
  { id: "google", name: "Google Gemini", icon: "💎" },
  { id: "ollama", name: "Ollama (Local)", icon: "🖥️" },
]

interface LanguageOption {
  code: string
  label: string
  native: string
}

const LANGUAGES: LanguageOption[] = [
  { code: "en", label: "English", native: "English" },
  { code: "zh", label: "中文", native: "中文" },
  { code: "es", label: "Español", native: "Español" },
]

function WelcomeStep({ onNext }: { onNext: () => void }) {
  const { t } = useTranslation()

  return (
    <div className="onboarding-step flex flex-col items-center justify-center gap-4">
      <img src="/logo_with_text.png" alt="OctAi" className="onboarding-logo" />
      <h2 className="onboarding-heading">{t("onboarding.welcome.title")}</h2>
      <p className="onboarding-description">{t("onboarding.welcome.description")}</p>
      <Button
        size="lg"
        onClick={onNext}
        className="w-full max-w-[200px] cursor-pointer"
        style={{ background: "#a855f7", color: "#fff" }}
      >
        {t("onboarding.welcome.getStarted")}
        <IconChevronRight size={18} />
      </Button>
    </div>
  )
}

function LanguageStep({ onNext }: { onNext: () => void }) {
  const { t, i18n } = useTranslation()
  const [selected, setSelected] = useState(i18n.language?.startsWith("zh") ? "zh" : i18n.language?.startsWith("es") ? "es" : "en")

  const handleChange = useCallback(
    (code: string) => {
      setSelected(code)
      i18n.changeLanguage(code)
    },
    [i18n],
  )

  return (
    <div className="onboarding-step flex flex-col">
      <div className="mb-6 flex items-center gap-2 justify-center">
        <IconLanguage size={24} className="text-[#a855f7]" />
        <h2 className="onboarding-heading !mb-0">{t("onboarding.language.title")}</h2>
      </div>
      <p className="onboarding-description">{t("onboarding.language.description")}</p>
      <div className="language-list">
        {LANGUAGES.map((lang) => (
          <button
            key={lang.code}
            type="button"
            className={`language-option ${selected === lang.code ? "selected" : ""}`}
            onClick={() => handleChange(lang.code)}
          >
            <span className="text-lg">{lang.code === "en" ? "🇺🇸" : lang.code === "zh" ? "🇨🇳" : "🇪🇸"}</span>
            <div className="flex flex-col items-start">
              <span className="language-option-label">{lang.label}</span>
              <span className="language-option-native">{lang.native}</span>
            </div>
            {selected === lang.code && <IconCheck size={18} className="ml-auto text-[#a855f7]" />}
          </button>
        ))}
      </div>
      <Button size="lg" onClick={onNext} className="mt-auto w-full cursor-pointer" style={{ background: "#a855f7", color: "#fff" }}>
        {t("onboarding.next")}
        <IconChevronRight size={18} />
      </Button>
    </div>
  )
}

function ProviderStep({ onNext }: { onNext: () => void }) {
  const { t } = useTranslation()
  const [selected, setSelected] = useState<string | null>(null)

  return (
    <div className="onboarding-step flex flex-col">
      <div className="mb-6 flex items-center gap-2 justify-center">
        <IconPlug size={24} className="text-[#a855f7]" />
        <h2 className="onboarding-heading !mb-0">{t("onboarding.provider.title")}</h2>
      </div>
      <p className="onboarding-description">{t("onboarding.provider.description")}</p>
      <div className="provider-grid">
        {PROVIDERS.map((provider) => (
          <button
            key={provider.id}
            type="button"
            className={`provider-card ${selected === provider.id ? "selected" : ""}`}
            onClick={() => setSelected(provider.id)}
          >
            <span className="provider-card-icon">{provider.icon}</span>
            <span className="provider-card-name">{provider.name}</span>
          </button>
        ))}
      </div>
      <Button size="lg" onClick={onNext} className="mt-auto w-full cursor-pointer" style={{ background: "#a855f7", color: "#fff" }}>
        {t("onboarding.next")}
        <IconChevronRight size={18} />
      </Button>
      <button type="button" className="mx-auto mt-2 text-sm text-[oklch(0.62_0.06_285)] hover:text-[oklch(0.95_0.01_280)] cursor-pointer transition-colors">
        {t("onboarding.provider.configureLater")}
      </button>
    </div>
  )
}

function MobileStep({ onNext, mobileEnabled, setMobileEnabled }: { onNext: () => void; mobileEnabled: boolean; setMobileEnabled: (v: boolean) => void }) {
  const { t } = useTranslation()

  return (
    <div className="onboarding-step flex flex-col">
      <div className="mb-6 flex items-center gap-2 justify-center">
        <IconDeviceMobile size={24} className="text-[#a855f7]" />
        <h2 className="onboarding-heading !mb-0">{t("onboarding.mobile.title")}</h2>
      </div>
      <p className="onboarding-description">{t("onboarding.mobile.description")}</p>
      <div className="mobile-toggle-row">
        <div>
          <div className="text-sm font-medium">{t("onboarding.mobile.enableNow")}</div>
          <div className="text-xs text-[oklch(0.62_0.06_285)]">{t("onboarding.mobile.enableNowDesc")}</div>
        </div>
        <Switch checked={mobileEnabled} onCheckedChange={setMobileEnabled} />
      </div>
      {mobileEnabled && (
        <div className="mobile-info">
          <IconServer size={16} className="inline mr-1 text-[#a855f7]" />
          {t("onboarding.mobile.tailscaleInfo")}
        </div>
      )}
      <Button size="lg" onClick={onNext} className="mt-auto w-full cursor-pointer" style={{ background: "#a855f7", color: "#fff" }}>
        {t("onboarding.next")}
        <IconChevronRight size={18} />
      </Button>
    </div>
  )
}

function CompleteStep({ language, provider, mobileEnabled, onFinish }: { language: string; provider: string | null; mobileEnabled: boolean; onFinish: () => void }) {
  const { t } = useTranslation()

  const languageName = LANGUAGES.find((l) => l.code === language)?.native ?? language
  const providerName = PROVIDERS.find((p) => p.id === provider)?.name ?? null

  return (
    <div className="onboarding-step flex flex-col items-center">
      <div className="complete-checkmark">
        <IconCheck size={32} />
      </div>
      <h2 className="onboarding-heading">{t("onboarding.complete.title")}</h2>
      <p className="onboarding-description">{t("onboarding.complete.description")}</p>
      <div className="complete-summary w-full">
        <div className="complete-summary-item">
          <span className="complete-summary-label">{t("onboarding.complete.language")}</span>
          <span className="complete-summary-value">{languageName}</span>
        </div>
        <div className="complete-summary-item">
          <span className="complete-summary-label">{t("onboarding.complete.provider")}</span>
          <span className="complete-summary-value">{providerName ?? t("onboarding.complete.notConfigured")}</span>
        </div>
        <div className="complete-summary-item">
          <span className="complete-summary-label">{t("onboarding.complete.mobile")}</span>
          <span className="complete-summary-value">{mobileEnabled ? t("onboarding.complete.enabled") : t("onboarding.complete.setupLater")}</span>
        </div>
      </div>
      <Button size="lg" onClick={onFinish} className="w-full cursor-pointer" style={{ background: "#a855f7", color: "#fff" }}>
        <IconRocket size={18} />
        {t("onboarding.complete.launch")}
      </Button>
    </div>
  )
}

export function OnboardingWizard() {
  const { i18n } = useTranslation()
  const [step, setStep] = useState(0)
  const [exiting, setExiting] = useState(false)
  const [mobileEnabled, setMobileEnabled] = useState(false)
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const handleNext = useCallback(() => {
    if (step === 1) {
      setSelectedProvider(null)
    }
    setExiting(true)
    setTimeout(() => {
      setStep((s) => Math.min(s + 1, TOTAL_STEPS - 1))
      setExiting(false)
    }, 200)
  }, [step])

  const handleBack = useCallback(() => {
    if (step === 0) return
    setExiting(true)
    setTimeout(() => {
      setStep((s) => s - 1)
      setExiting(false)
    }, 200)
  }, [step])

  const handleFinish = useCallback(() => {
    localStorage.setItem(ONBOARDING_KEY, "true")
    setExiting(true)
    setTimeout(() => {
      setStep(TOTAL_STEPS)
    }, 200)
  }, [])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        handleFinish()
      } else if (e.key === "Enter") {
        if (step < TOTAL_STEPS - 1) {
          handleNext()
        } else {
          handleFinish()
        }
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [step, handleNext, handleFinish])

  if (step >= TOTAL_STEPS) return null

  const currentLanguage = i18n.language?.startsWith("zh") ? "zh" : i18n.language?.startsWith("es") ? "es" : "en"

  return (
    <div className="onboarding-overlay" ref={containerRef}>
      <div className="onboarding-container">
        <div className="onboarding-progress">
          {Array.from({ length: TOTAL_STEPS }, (_, i) => (
            <div key={i} className={`onboarding-progress-dot ${i === step ? "active" : ""} ${i < step ? "completed" : ""}`} />
          ))}
        </div>

        <div className="onboarding-body">
          <div className={exiting ? "onboarding-step-exit" : ""}>
            {step === 0 && <WelcomeStep onNext={handleNext} />}
            {step === 1 && <LanguageStep onNext={handleNext} />}
            {step === 2 && <ProviderStep onNext={handleNext} />}
            {step === 3 && <MobileStep onNext={handleNext} mobileEnabled={mobileEnabled} setMobileEnabled={setMobileEnabled} />}
            {step === 4 && <CompleteStep language={currentLanguage} provider={selectedProvider} mobileEnabled={mobileEnabled} onFinish={handleFinish} />}
          </div>
        </div>

        {step > 0 && step < TOTAL_STEPS - 1 && (
          <div className="onboarding-footer">
            <Button variant="ghost" size="sm" onClick={handleBack} className="cursor-pointer">
              <IconChevronLeft size={16} />
            </Button>
            <span className="text-xs text-[oklch(0.50_0.05_280)]">
              {step} / {TOTAL_STEPS - 1}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}
