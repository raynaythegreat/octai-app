import { Outlet, createRootRoute } from "@tanstack/react-router"
import { useEffect, useState } from "react"

import { AppLayout } from "@/components/app-layout"
import { OnboardingWizard } from "@/components/onboarding/onboarding-wizard"
import { initializeChatStore } from "@/features/chat/controller"

const ONBOARDING_KEY = "octai-onboarded"

const RootLayout = () => {
  const [showOnboarding] = useState(() => localStorage.getItem(ONBOARDING_KEY) !== "true")

  useEffect(() => {
    initializeChatStore()
  }, [])

  return (
    <AppLayout>
      <Outlet />
      {showOnboarding && <OnboardingWizard />}
    </AppLayout>
  )
}

export const Route = createRootRoute({ component: RootLayout })
