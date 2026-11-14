// MedusaGo has no AI assistant: a stub keeping the upstream interface.
import React from "react"

export type AiAssistantContextType = {
  chatOpened: boolean
  setChatOpened: (value: boolean) => void
  loading: boolean
  [key: string]: unknown
}

const noop = () => {}
const value: AiAssistantContextType = {
  chatOpened: false,
  setChatOpened: noop,
  loading: false,
}

export const AiAssistantProvider = ({
  children,
}: {
  children?: React.ReactNode
  integrationId?: string
}) => <>{children}</>

export const useAiAssistant = () => value
